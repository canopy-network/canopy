package crypto

import (
	// This is the VDF used in Harmony Project. It is based on Benjanmin Wesolowski's paper "Efficient verifiable delay functions"(https://eprint.iacr.org/2018/623.pdf)
	"github.com/harmony-one/vdf/src/vdf_go"
	"sync/atomic"
	"time"
)

const (
	// StartingIterationsPerSecond is a 'best-guess' estimate based on 2.3 GHz 8-Core Intel Core i9
	StartingIterationsPerSecond = 550
	// IterationsFixedDecreasePercent is a 'best-guess' iteration reduction when a Stop() is called before finishing
	IterationsFixedDecreasePercent = float64(10)
	// BitSize of 2048 is required by the harmony-one VDF library
	BitSize = 2048
)

type VDFService struct {
	TargetTime time.Duration // target time should have a pre-baked in 'breathing room' to prevent misses due to accidental overages
	Iterations int           // number of iterations the VDF will currently Run()
	Output     []byte        // the output from the previous VDF run
	stopChan   chan struct{}
	running    *atomic.Bool
}

// NewVDFService() creates a new instance of the VDF service
func NewVDFService(targetTime time.Duration) (vdf *VDFService) {
	vdf = &VDFService{TargetTime: targetTime, stopChan: make(chan struct{}), running: &atomic.Bool{}}
	vdf.estimateIterationsPerSecond()
	return
}

// Run() generates a VDF proof using the current params state of the VDF Service object
func (vdf *VDFService) Run(seed []byte) {
	if vdf == nil {
		return
	}
	// - Run() and not running locks and starts a run
	// - Run() and already running returns
	if !vdf.running.CompareAndSwap(false, true) {
		return
	}
	// reset the sync variable
	defer vdf.running.Store(false)
	// track the start time to measure the 'processing time'
	startTime := time.Now()
	// run the VDF generation - if Stop() called, this will exit prematurely with y and proof being nil
	y, proof := vdf_go.GenerateVDFWithStopChan(seed, vdf.Iterations, BitSize, vdf.stopChan)
	// adjusting variables so must lock for thread safety as the Stop() function may be accessing the `Output`
	// if prematurely stopped
	if y == nil || proof == nil {
		// don't know how long was left in the VDF so decrease iterations by a fixed amount
		// example: 10% fixed decrease on 500 iterations = 450 next iterations
		vdf.Iterations = int(float64(vdf.Iterations) * (1 - IterationsFixedDecreasePercent/100))
		// exit
		return
	}
	// combine the y and proof as that's how it's verified
	vdf.Output = append(y, proof...)
	// adjust the iterations based on completion time
	vdf.adjustIterations(time.Since(startTime))
}

// Finish() signals the service to complete and returns the output
// - already running signals a stop in the running thread and returns
// - not running returns
func (vdf *VDFService) Finish() (out []byte, iterations int) {
	if vdf == nil {
		return
	}
	// if service has not yet completed, signal to stop
	if vdf.running.Load() {
		vdf.stopChan <- struct{}{} // NOTE: multiple sequential calls to stop is not supported
		return
	}
	if vdf.Output == nil {
		return
	}
	return vdf.Output, vdf.Iterations
}

// VerifyVDF() verifies the VDF using the seed, the proof, and the number of iterations
func (vdf *VDFService) VerifyVDF(seed, proof []byte, iterations int) bool {
	return vdf_go.VerifyVDF(seed, proof, iterations, BitSize)
}

// estimateIterationsPerSecond() runs a quick VDF test to determine what the iterations per second is on this processor
// NOTE: longer target times have been observed to complete more iterations quicker,
// so theoretically this a safe starting place
func (vdf *VDFService) estimateIterationsPerSecond() {
	var (
		totalTime  time.Duration
		iterations = 5 // more iterations  = longer startup time
	)
	for i := 0; i < iterations; i++ {
		startTime := time.Now()
		_, _ = vdf_go.GenerateVDF(nil, StartingIterationsPerSecond, BitSize)
		totalTime += time.Since(startTime)
	}
	// calculate average seconds per iteration
	averageSeconds := totalTime.Seconds() / float64(iterations)
	// set the iterations number based on the deviation from 1 second
	vdf.Iterations = int(float64(StartingIterationsPerSecond) / averageSeconds)
}

// adjustIterations() changes the number of iterations to be completed based on the
// previous result and the target time
func (vdf *VDFService) adjustIterations(actualTime time.Duration) {
	// coefficient = target_time / actual_time
	adjustmentCoefficient := vdf.TargetTime.Seconds() / actualTime.Seconds()
	// new_iterations = old_iterations * coefficient
	vdf.Iterations = int(float64(vdf.Iterations) * adjustmentCoefficient)
}