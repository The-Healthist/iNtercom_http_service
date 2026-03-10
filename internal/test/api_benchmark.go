package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// APIBenchmark 定义API基准测试结构
type APIBenchmark struct {
	BaseURL     string
	Concurrency int
	Requests    int
	AuthToken   string
	Client      *http.Client
}

// BenchmarkResult 定义基准测试结果
type BenchmarkResult struct {
	URL            string        `json:"url"`
	Method         string        `json:"method"`
	Concurrency    int           `json:"concurrency"`
	TotalRequests  int           `json:"total_requests"`
	SuccessCount   int           `json:"success_count"`
	FailureCount   int           `json:"failure_count"`
	TotalTime      time.Duration `json:"total_time"`
	AverageTime    time.Duration `json:"average_time"`
	MinTime        time.Duration `json:"min_time"`
	MaxTime        time.Duration `json:"max_time"`
	RequestsPerSec float64       `json:"requests_per_sec"`
	StatusCodes    map[int]int   `json:"status_codes"`
	Errors         []string      `json:"errors"`
}

// RequestResult 定义单个请求的结果
type RequestResult struct {
	Duration   time.Duration
	StatusCode int
	Error      error
}

// NewAPIBenchmark 创建新的API基准测试实例
func NewAPIBenchmark(baseURL string, concurrency, requests int, authToken string) *APIBenchmark {
	return &APIBenchmark{
		BaseURL:     baseURL,
		Concurrency: concurrency,
		Requests:    requests,
		AuthToken:   authToken,
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// RunGET 执行GET请求的基准测试
func (b *APIBenchmark) RunGET(path string) *BenchmarkResult {
	url := b.BaseURL + path
	return b.runTest(http.MethodGet, url, nil)
}

// RunPOST 执行POST请求的基准测试
func (b *APIBenchmark) RunPOST(path string, payload interface{}) *BenchmarkResult {
	url := b.BaseURL + path
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return &BenchmarkResult{
			URL:    url,
			Method: http.MethodPost,
			Errors: []string{fmt.Sprintf("JSON编码错误: %v", err)},
		}
	}
	return b.runTest(http.MethodPost, url, jsonData)
}

// RunPUT 执行PUT请求的基准测试
func (b *APIBenchmark) RunPUT(path string, payload interface{}) *BenchmarkResult {
	url := b.BaseURL + path
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return &BenchmarkResult{
			URL:    url,
			Method: http.MethodPut,
			Errors: []string{fmt.Sprintf("JSON编码错误: %v", err)},
		}
	}
	return b.runTest(http.MethodPut, url, jsonData)
}

// RunDELETE 执行DELETE请求的基准测试
func (b *APIBenchmark) RunDELETE(path string) *BenchmarkResult {
	url := b.BaseURL + path
	return b.runTest(http.MethodDelete, url, nil)
}

// runTest 执行基准测试
func (b *APIBenchmark) runTest(method, url string, payload []byte) *BenchmarkResult {
	results := make(chan RequestResult, b.Requests)
	var wg sync.WaitGroup
	limiter := make(chan struct{}, b.Concurrency)

	startTime := time.Now()

	// 创建工作池
	for i := 0; i < b.Requests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			limiter <- struct{}{}
			defer func() { <-limiter }()

			start := time.Now()
			req, err := http.NewRequest(method, url, bytes.NewBuffer(payload))
			if err != nil {
				results <- RequestResult{Error: err}
				return
			}

			req.Header.Set("Content-Type", "application/json")
			if b.AuthToken != "" {
				req.Header.Set("Authorization", "Bearer "+b.AuthToken)
			}

			resp, err := b.Client.Do(req)
			if err != nil {
				results <- RequestResult{Error: err}
				return
			}
			defer resp.Body.Close()

			results <- RequestResult{
				Duration:   time.Since(start),
				StatusCode: resp.StatusCode,
			}
		}()
	}

	// 等待所有请求完成
	go func() {
		wg.Wait()
		close(results)
	}()

	// 收集结果
	var minTime time.Duration = 1<<63 - 1
	var maxTime time.Duration
	var totalTime time.Duration
	successCount := 0
	failureCount := 0
	statusCodes := make(map[int]int)
	var errors []string

	for result := range results {
		if result.Error != nil {
			failureCount++
			errors = append(errors, result.Error.Error())
			continue
		}

		totalTime += result.Duration
		if result.Duration < minTime {
			minTime = result.Duration
		}
		if result.Duration > maxTime {
			maxTime = result.Duration
		}

		statusCodes[result.StatusCode]++
		if result.StatusCode >= 200 && result.StatusCode < 300 {
			successCount++
		} else {
			failureCount++
		}
	}

	totalElapsed := time.Since(startTime)
	requestsPerSec := float64(b.Requests) / totalElapsed.Seconds()
	averageTime := time.Duration(0)
	if successCount+failureCount > 0 {
		averageTime = totalTime / time.Duration(successCount+failureCount)
	}

	return &BenchmarkResult{
		URL:            url,
		Method:         method,
		Concurrency:    b.Concurrency,
		TotalRequests:  b.Requests,
		SuccessCount:   successCount,
		FailureCount:   failureCount,
		TotalTime:      totalElapsed,
		AverageTime:    averageTime,
		MinTime:        minTime,
		MaxTime:        maxTime,
		RequestsPerSec: requestsPerSec,
		StatusCodes:    statusCodes,
		Errors:         errors,
	}
}

// PrintResult 打印基准测试结果
func (r *BenchmarkResult) PrintResult() {
	fmt.Printf("基准测试结果:\n")
	fmt.Printf("URL: %s\n", r.URL)
	fmt.Printf("方法: %s\n", r.Method)
	fmt.Printf("并发数: %d\n", r.Concurrency)
	fmt.Printf("总请求数: %d\n", r.TotalRequests)
	fmt.Printf("成功请求数: %d\n", r.SuccessCount)
	fmt.Printf("失败请求数: %d\n", r.FailureCount)
	fmt.Printf("总耗时: %s\n", r.TotalTime)
	fmt.Printf("平均耗时: %s\n", r.AverageTime)
	fmt.Printf("最小耗时: %s\n", r.MinTime)
	fmt.Printf("最大耗时: %s\n", r.MaxTime)
	fmt.Printf("每秒请求数: %.2f\n", r.RequestsPerSec)
	fmt.Printf("状态码分布:\n")
	for code, count := range r.StatusCodes {
		fmt.Printf("  %d: %d\n", code, count)
	}
	if len(r.Errors) > 0 {
		fmt.Printf("错误信息 (最多显示5个):\n")
		for i, err := range r.Errors {
			if i >= 5 {
				fmt.Printf("  ... 还有 %d 个错误\n", len(r.Errors)-5)
				break
			}
			fmt.Printf("  %s\n", err)
		}
	}
}
