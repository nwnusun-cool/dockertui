# 容器资源监控历史趋势功能

## 功能概述

将容器详情页面的资源监控从"实时快照"扩展为"短期历史趋势"，支持用户查看过去一段时间的资源使用情况，并可以切换不同的时间粒度。

## 设计思路

### 1. 数据采集与存储

**关键约束**：Docker API 不提供历史数据，只返回当前时刻的资源使用情况。

**解决方案**：
- 在 docktui 运行期间，每秒采样一次容器资源数据
- 将原始数据存储在内存中（最多保存 30 分钟）
- 根据用户选择的时间粒度，动态聚合原始数据

**数据结构**：
```go
type DataPoint struct {
    Timestamp time.Time
    Value     float64
}

type StatsView struct {
    // 原始数据（1秒采样）
    cpuRawData    []DataPoint  // 最多 1800 个点（30分钟）
    memoryRawData []DataPoint
    
    // 聚合后的数据（用于显示）
    cpuHistory    []float64    // 固定 60 个点
    memoryHistory []float64
    
    // 当前时间粒度
    granularity TimeGranularity
}
```

### 2. 时间粒度设计

**优化后的粒度**（更符合实际使用场景）：

| 粒度 | 间隔 | 显示范围 | 数据点数 | 适用场景 |
|------|------|----------|----------|----------|
| 1秒  | 1s   | 最近1分钟 | 60 | 实时监控，查看瞬时波动 |
| 5秒  | 5s   | 最近5分钟 | 60 | 短期趋势，查看最近变化 |
| 10秒 | 10s  | 最近10分钟 | 60 | 中期趋势，查看稳定性 |
| 30秒 | 30s  | 最近30分钟 | 60 | 长期趋势，查看整体情况 |

**为什么选择这些粒度？**
1. **数据来源限制**：只有 docktui 运行期间采集的数据
2. **实际使用场景**：用户通常只需要查看最近几分钟到半小时的数据
3. **内存占用**：30 分钟 × 60 秒 = 1800 个数据点，内存占用很小
4. **响应速度**：数据量小，聚合速度快

### 3. 数据聚合算法

将原始数据（1秒采样）聚合为固定 60 个点：

```go
func aggregateDataPoints(data []DataPoint, interval time.Duration, maxPoints int) []float64 {
    result := make([]float64, 0, maxPoints)
    now := time.Now()
    startTime := now.Add(-time.Duration(maxPoints) * interval)
    
    for i := 0; i < maxPoints; i++ {
        bucketStart := startTime.Add(time.Duration(i) * interval)
        bucketEnd := bucketStart.Add(interval)
        
        // 收集该时间段内的所有数据点
        var sum float64
        var count int
        for _, point := range data {
            if point.Timestamp.After(bucketStart) && point.Timestamp.Before(bucketEnd) {
                sum += point.Value
                count++
            }
        }
        
        // 计算平均值
        if count > 0 {
            result = append(result, sum/float64(count))
        } else {
            // 没有数据点，使用前一个值或 0
            if len(result) > 0 {
                result = append(result, result[len(result)-1])
            } else {
                result = append(result, 0)
            }
        }
    }
    
    return result
}
```

### 4. 用户交互

**快捷键**：
- `1`: 切换到 1秒 粒度（最近1分钟）
- `2`: 切换到 5秒 粒度（最近5分钟）
- `3`: 切换到 10秒 粒度（最近10分钟）
- `4`: 切换到 30秒 粒度（最近30分钟）

**视觉反馈**：
- 时间粒度选择高亮显示当前选中的粒度
- 图表标题动态显示时间范围（如"CPU 使用率 (最近5分钟)"）
- 图表数据实时更新

## 实现细节

### 1. 数据采集

在 `StatsView.Update` 中处理 `statsLoadedMsg`：
```go
case statsLoadedMsg:
    v.updateStats(msg.stats)
    if v.active {
        return v.scheduleRefresh()  // 每秒刷新一次
    }
```

### 2. 数据存储

在 `updateStats` 中添加原始数据：
```go
now := time.Now()
v.cpuRawData = append(v.cpuRawData, DataPoint{
    Timestamp: now,
    Value:     stats.CPUPercent,
})

// 清理过期数据（保留最近30分钟）
cutoff := now.Add(-30 * time.Minute)
v.cpuRawData = v.cleanOldData(v.cpuRawData, cutoff)
```

### 3. 时间粒度切换

在 `StatsView.Update` 中处理数字键：
```go
case tea.KeyMsg:
    switch msg.String() {
    case "1":
        v.setGranularity(Granularity1s)
    case "2":
        v.setGranularity(Granularity5s)
    case "3":
        v.setGranularity(Granularity10s)
    case "4":
        v.setGranularity(Granularity30s)
    }
```

### 4. 数据聚合

在 `setGranularity` 中触发重新聚合：
```go
func (v *StatsView) setGranularity(g TimeGranularity) {
    v.granularity = g
    v.aggregateData()  // 重新聚合数据并更新图表
}
```

## 优势

1. **实时性**：数据每秒更新，反映最新状态
2. **灵活性**：支持多种时间粒度，满足不同需求
3. **轻量级**：只保存 30 分钟数据，内存占用小
4. **响应快**：数据量小，聚合速度快
5. **直观性**：图表标题和时间粒度选择提供清晰的视觉反馈

## 限制

1. **数据持久化**：关闭程序后数据丢失（可以考虑后续添加导出功能）
2. **历史范围**：最多查看 30 分钟历史（符合实际使用场景）
3. **采样精度**：1 秒采样，无法捕捉更细粒度的波动

## 未来扩展

1. **数据导出**：支持导出 CSV 格式的历史数据
2. **告警功能**：当资源使用超过阈值时提醒用户
3. **对比功能**：支持多个容器的资源使用对比
4. **持久化**：可选的本地数据库存储（如 SQLite）
