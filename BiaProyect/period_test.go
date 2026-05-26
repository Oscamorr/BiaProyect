package main

import (
	"testing"
	"time"
)

func TestBuildPeriodsDaily(t *testing.T) {
	start := time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2023, 6, 10, 0, 0, 0, 0, time.UTC).Add(24 * time.Hour)

	periods := buildPeriods(start, end, "daily")

	if len(periods) != 10 {
		t.Fatalf("esperaba 10 periodos diarios, obtuve %d", len(periods))
	}

	if periods[0].Label != "JUN 1" {
		t.Errorf("primer label incorrecto, got %s, want %s", periods[0].Label, "JUN 1")
	}

	if periods[9].Label != "JUN 10" {
		t.Errorf("ultimo label incorrecto, got %s, want %s", periods[9].Label, "JUN 10")
	}
}

func TestBuildPeriodsWeekly(t *testing.T) {
	start := time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2023, 6, 26, 0, 0, 0, 0, time.UTC).Add(24 * time.Hour)

	periods := buildPeriods(start, end, "weekly")

	if len(periods) != 4 {
		t.Fatalf("esperaba 4 periodos semanales, obtuve %d", len(periods))
	}

	if periods[0].Label != "JUN 1 - JUN 7" {
		t.Errorf("primer label incorrecto, got %s, want %s", periods[0].Label, "JUN 1 - JUN 7")
	}
}

func TestBuildPeriodsMonthly(t *testing.T) {
	start := time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2023, 8, 10, 0, 0, 0, 0, time.UTC).Add(24 * time.Hour)

	periods := buildPeriods(start, end, "monthly")

	if len(periods) < 2 {
		t.Fatalf("esperaba al menos 2 periodos mensuales, obtuve %d", len(periods))
	}

	if periods[0].Label != "JUN 2023" {
		t.Errorf("primer label incorrecto, got %s, want %s", periods[0].Label, "JUN 2023")
	}
}
