package audio

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
)

// BatchResult represents the result of processing a single file in a batch
type BatchResult struct {
	FilePath string
	Error    error
	Analysis *AudioAnalysis
}

// BatchProgress represents the progress of batch processing
type BatchProgress struct {
	TotalFiles     int
	ProcessedFiles int
	CurrentFile    string
	Results        []BatchResult
}

// BatchProcessor handles processing of multiple audio files
type BatchProcessor struct {
	analyzer   *AudioAnalyzer
	processor  *FFmpegProcessor
	maxWorkers int
	progressCh chan BatchProgress
	results    []BatchResult
	mu         sync.Mutex
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(analyzer *AudioAnalyzer, processor *FFmpegProcessor, maxWorkers int) *BatchProcessor {
	if maxWorkers <= 0 {
		maxWorkers = 4 // Default to 4 workers
	}
	return &BatchProcessor{
		analyzer:   analyzer,
		processor:  processor,
		maxWorkers: maxWorkers,
		progressCh: make(chan BatchProgress, 1),
		results:    make([]BatchResult, 0),
	}
}

// ProcessFiles processes multiple audio files in parallel
func (b *BatchProcessor) ProcessFiles(ctx context.Context, files []string, opts ProcessingOptions) (<-chan BatchProgress, error) {
	if len(files) == 0 {
		return nil, fmt.Errorf("no files to process")
	}

	// Create a buffered channel for work items
	workCh := make(chan string, len(files))
	for _, file := range files {
		workCh <- file
	}
	close(workCh)

	// Create worker pool
	var wg sync.WaitGroup
	for i := 0; i < b.maxWorkers; i++ {
		wg.Add(1)
		go b.worker(ctx, &wg, workCh, opts)
	}

	// Start progress monitoring in a separate goroutine
	go func() {
		wg.Wait()
		close(b.progressCh)
	}()

	return b.progressCh, nil
}

// worker processes files from the work channel
func (b *BatchProcessor) worker(ctx context.Context, wg *sync.WaitGroup, workCh <-chan string, opts ProcessingOptions) {
	defer wg.Done()

	for filePath := range workCh {
		select {
		case <-ctx.Done():
			return
		default:
			result := b.processFile(ctx, filePath, opts)
			b.addResult(result)
			b.updateProgress(filePath)
		}
	}
}

// processFile processes a single file
func (b *BatchProcessor) processFile(ctx context.Context, filePath string, opts ProcessingOptions) BatchResult {
	result := BatchResult{
		FilePath: filePath,
	}

	// Process the file
	outputPath := filepath.Join(filepath.Dir(filePath), "processed_"+filepath.Base(filePath))
	if err := b.processor.ProcessAudio(ctx, filePath, outputPath, opts); err != nil {
		result.Error = fmt.Errorf("processing failed: %w", err)
		return result
	}

	// Analyze the processed file
	analysis, err := b.analyzer.AnalyzeTrack(ctx, outputPath)
	if err != nil {
		result.Error = fmt.Errorf("analysis failed: %w", err)
		return result
	}
	result.Analysis = analysis

	return result
}

// addResult adds a result to the results slice thread-safely
func (b *BatchProcessor) addResult(result BatchResult) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.results = append(b.results, result)
}

// updateProgress sends a progress update
func (b *BatchProcessor) updateProgress(currentFile string) {
	b.mu.Lock()
	progress := BatchProgress{
		TotalFiles:     len(b.results),
		ProcessedFiles: len(b.results),
		CurrentFile:    currentFile,
		Results:        make([]BatchResult, len(b.results)),
	}
	copy(progress.Results, b.results)
	b.mu.Unlock()

	// Send progress update non-blocking
	select {
	case b.progressCh <- progress:
	default:
		// Channel is full, skip this update
	}
}

// GetResults returns all processing results
func (b *BatchProcessor) GetResults() []BatchResult {
	b.mu.Lock()
	defer b.mu.Unlock()
	results := make([]BatchResult, len(b.results))
	copy(results, b.results)
	return results
}

// ClearResults clears all processing results
func (b *BatchProcessor) ClearResults() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.results = b.results[:0]
}
