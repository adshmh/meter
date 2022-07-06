package collector

import (
	"context"
	"fmt"
	"time"

	logger "github.com/sirupsen/logrus"

	"github.com/adshmh/meter/api"
)

const (
	COLLECT_INTERVAL_SECONDS = 120
	REPORT_INTERVAL_SECONDS  = 10
)

type Source interface {
	DailyCounts(from, to time.Time) (map[time.Time]map[string]int64, error)
	TodaysCounts() (map[string]int64, error)
}

type Writer interface {
	// Returns the 2 timestamps which mark the first and last day for
	//	which the metrics are saved.
	//	It is assumed that there are no gaps in the returned time period.
	ExistingMetricsTimespan() (time.Time, time.Time, error)
	// TODO: allow overwriting today's metrics
	WriteDailyUsage(counts map[time.Time]map[string]int64) error
	WriteTodaysUsage(counts map[string]int64) error
}

type Collector interface {
	// Start a goroutine, which collects data at set intervals
	//	The routine respects existing metrics, i.e. will not collect/overwrite existing metrics
	//	expect for today's metrics
	Start(ctx context.Context, collectIntervalSeconds, reportIntervalSeconds int)
	// Collect and write metrics data: this will overwrite any existing metrics
	//	This function exists to allow manually overriding the collector's behavior.
	Collect(from, to time.Time) error
}

// NewCollector returns a collector which will periodically (or on Collect being called)
//	gathers metrics from the source and writes to the writer.
//	maxArchiveAge is the oldest time for which metrics are saved
func NewCollector(source Source, writer Writer, maxArchiveAge time.Duration, log *logger.Logger) Collector {
	return &collector{
		Source:        source,
		Writer:        writer,
		MaxArchiveAge: maxArchiveAge,
		Logger:        log,
	}
}

type collector struct {
	Source
	Writer
	MaxArchiveAge time.Duration
	*logger.Logger
}

// Collects relay usage data from the source and uses the writer to store.
//	-
func (c *collector) Collect(from, to time.Time) error {
	from, to, err := api.AdjustTimePeriod(from, time.Now())
	if err != nil {
		return err
	}

	counts, err := c.Source.DailyCounts(from, to)
	if err != nil {
		return err
	}
	if err := c.Writer.WriteDailyUsage(counts); err != nil {
		return err
	}

	todaysCounts, err := c.Source.TodaysCounts()
	if err != nil {
		return err
	}

	return c.Writer.WriteTodaysUsage(todaysCounts)
}

func (c *collector) collect() error {
	first, last, err := c.Writer.ExistingMetricsTimespan()
	if err != nil {
		return err
	}

	// We assume there are no gaps between stored metrics from start to end, so
	// 	start collecting metrics after the last saved date
	var from time.Time
	if first.Equal(time.Time{}) {
		from = time.Now().Add(-1 * c.MaxArchiveAge)
	} else {
		dayLayout := "2006-01-02"
		today, err := time.Parse(dayLayout, time.Now().Format(dayLayout))
		if err != nil {
			return err
		}

		from = last.AddDate(0, 0, 1)
		if from.After(today) {
			from = today
		}
	}

	return c.Collect(from, time.Now())
}

func (c *collector) Start(ctx context.Context, collectIntervalSeconds, reportIntervalSeconds int) {
	// Do an initial data collection, and then repeat on set intervals
	if err := c.collect(); err != nil {
		c.Logger.WithFields(logger.Fields{"error": err}).Warn("Failed to collect data")
	}

	for {
		remaining := collectIntervalSeconds // COLLECT_INTERVAL_SECONDS
		for {
			select {
			case <-ctx.Done():
				c.Logger.Warn("Context has been cancelled. Collecter exiting.")
				return
			case <-time.After(time.Second * time.Duration(reportIntervalSeconds)):
				remaining -= reportIntervalSeconds
				if remaining > 0 {
					c.Logger.Info(fmt.Sprintf("Will collect data in %d seconds...", remaining))
				} else {
					if err := c.collect(); err != nil {
						c.Logger.WithFields(logger.Fields{"error": err}).Warn("Failed to collect data")
					}
					break
				}
			}
		}
	}
}
