package api

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	logger "github.com/sirupsen/logrus"

	"github.com/pokt-foundation/portal-api-go/repository"
)

func TestUserRelays(t *testing.T) {
	now, _ := time.Parse(dayFormat, time.Now().Format(dayFormat))
	usageData := fakeDailyMetrics()
	todaysUsage := fakeTodaysMetrics()

	testCases := []struct {
		name     string
		user     string
		from     time.Time
		to       time.Time
		expected UserRelaysResponse
	}{
		{
			name: "Correct summary for a user",
			user: "user1",
			from: now.AddDate(0, 0, -6),
			to:   now,
			expected: UserRelaysResponse{
				From:         now.AddDate(0, 0, -6),
				To:           now.AddDate(0, 0, 1),
				User:         "user1",
				Applications: []string{"app1", "app2", "app3"},
				Count: RelayCounts{
					Success: 6*(2+1) + 50 + 30,
					Failure: 6*(3+5) + 40 + 70,
				},
			},
		},
		{
			name: "Correct summary for a user excluding today",
			user: "user1",
			from: now.AddDate(0, 0, -6),
			to:   now.AddDate(0, 0, -2),
			expected: UserRelaysResponse{
				From:         now.AddDate(0, 0, -6),
				To:           now.AddDate(0, 0, -1),
				User:         "user1",
				Applications: []string{"app1", "app2", "app3"},
				Count: RelayCounts{
					Success: 5 * (2 + 1),
					Failure: 5 * (3 + 5),
				},
			},
		},
		{
			name: "Correct summary for a user on todays metrics",
			user: "user1",
			from: now,
			to:   now,
			expected: UserRelaysResponse{
				From:         now,
				To:           now.AddDate(0, 0, 1),
				User:         "user1",
				Applications: []string{"app1", "app2", "app3"},
				Count: RelayCounts{
					Success: 50 + 30,
					Failure: 40 + 70,
				},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fakeBackend := fakeBackend{
				usage:       usageData,
				todaysUsage: todaysUsage,
				userApps: map[string][]string{
					"user1": {"app1", "app2", "app3"},
					"user2": {"app4", "app5", "app6"},
				},
			}

			relayMeter := NewRelayMeter(&fakeBackend, logger.New(), RelayMeterOptions{LoadInterval: 100 * time.Millisecond})
			time.Sleep(200 * time.Millisecond)
			got, err := relayMeter.UserRelays(tc.user, tc.from, tc.to)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if diff := cmp.Diff(tc.expected, got); diff != "" {
				t.Errorf("unexpected value (-want +got):\n%s", diff)
			}
		})
	}
}

func TestTotalRelays(t *testing.T) {
	now, _ := time.Parse(dayFormat, time.Now().Format(dayFormat))
	usageData := fakeDailyMetrics()
	todaysUsage := fakeTodaysMetrics()

	testCases := []struct {
		name     string
		from     time.Time
		to       time.Time
		expected TotalRelaysResponse
	}{
		{
			name: "Correct summary for the network",
			from: now.AddDate(0, 0, -6),
			to:   now,
			expected: TotalRelaysResponse{
				From: now.AddDate(0, 0, -6),
				To:   now.AddDate(0, 0, 1),
				Count: RelayCounts{
					Success: 6*(2+1+5) + 50 + 30 + 500,
					Failure: 6*(3+5+7) + 40 + 70 + 700,
				},
			},
		},
		{
			name: "Correct summary for the network excluding today",
			from: now.AddDate(0, 0, -6),
			to:   now.AddDate(0, 0, -2),
			expected: TotalRelaysResponse{
				From: now.AddDate(0, 0, -6),
				To:   now.AddDate(0, 0, -1),
				Count: RelayCounts{
					Success: 5 * (2 + 1 + 5),
					Failure: 5 * (3 + 5 + 7),
				},
			},
		},
		{
			name: "Correct summary for the network on todays metrics",
			from: now,
			to:   now,
			expected: TotalRelaysResponse{
				From: now,
				To:   now.AddDate(0, 0, 1),
				Count: RelayCounts{
					Success: 50 + 30 + 500,
					Failure: 40 + 70 + 700,
				},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fakeBackend := fakeBackend{
				usage:       usageData,
				todaysUsage: todaysUsage,
			}

			relayMeter := NewRelayMeter(&fakeBackend, logger.New(), RelayMeterOptions{LoadInterval: 100 * time.Millisecond})
			time.Sleep(200 * time.Millisecond)
			got, err := relayMeter.TotalRelays(tc.from, tc.to)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if diff := cmp.Diff(tc.expected, got); diff != "" {
				t.Errorf("unexpected value (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAppRelays(t *testing.T) {
	now, _ := time.Parse(dayFormat, time.Now().Format(dayFormat))
	usageData := fakeDailyMetrics()
	todaysUsage := fakeTodaysMetrics()
	requestedApp := "app1"

	testCases := []struct {
		name                string
		from                time.Time
		to                  time.Time
		usageData           map[time.Time]map[string]int64
		todaysUsage         map[string]int64
		expected            AppRelaysResponse
		expectedErr         error
		expectedTodaysCalls int
	}{
		{
			name: "Correct count is returned",
			from: now.AddDate(0, 0, -5),
			to:   now.AddDate(0, 0, -1),
			expected: AppRelaysResponse{
				Application: requestedApp,
				From:        now.AddDate(0, 0, -5),
				To:          now,
				Count: RelayCounts{
					Success: 5 * 2,
					Failure: 5 * 3,
				},
			},
		},
		{
			name: "From and To parameters are adjusted to start of the specifed day and the next day, respectively",
			from: time.Now().AddDate(0, 0, -5),
			to:   time.Now().AddDate(0, 0, -1),
			expected: AppRelaysResponse{
				Application: "app1",
				From:        now.AddDate(0, 0, -5),
				To:          now,
				Count: RelayCounts{
					Success: 5 * 2,
					Failure: 5 * 3,
				},
			},
		},
		{
			name: "Equal values are allowed for From and To parameters (to cover a single day)",
			from: now.AddDate(0, 0, -3),
			to:   now.AddDate(0, 0, -3),
			expected: AppRelaysResponse{
				Application: "app1",
				From:        now.AddDate(0, 0, -3),
				To:          now.AddDate(0, 0, -2),
				Count: RelayCounts{
					Success: 1 * 2,
					Failure: 1 * 3,
				},
			},
		},
		{
			name: "Today's metrics are added to previous days' usage data",
			from: now.AddDate(0, 0, -3),
			to:   now,
			todaysUsage: map[string]int64{
				"app1": 50,
				"app2": 30,
			},
			expected: AppRelaysResponse{
				Application: "app1",
				From:        now.AddDate(0, 0, -3),
				To:          now.AddDate(0, 0, 1),
				Count: RelayCounts{
					Success: 3*2 + 50,
					Failure: 3*3 + 40,
				},
			},
		},
		{
			name: "Only today's data is included when from and to parameters point to today",
			from: now,
			to:   now,
			expected: AppRelaysResponse{
				Application: "app1",
				From:        now,
				To:          now.AddDate(0, 0, 1),
				Count: RelayCounts{
					Success: 50,
					Failure: 40,
				},
			},
		},
		{
			name: "Today's metrics are not included when the 'to' parameter does not include today",
			from: time.Now().AddDate(0, 0, -3),
			to:   time.Now().AddDate(0, 0, -1),
			expected: AppRelaysResponse{
				Application: "app1",
				From:        now.AddDate(0, 0, -3),
				To:          now,
				Count: RelayCounts{
					Success: 3 * 2,
					Failure: 3 * 3,
				},
			},
		},
		{
			name: "Today's metrics are included when the 'to' parameter is after today",
			from: time.Now().AddDate(0, 0, -3),
			to:   time.Now().AddDate(0, 0, 2),
			expected: AppRelaysResponse{
				Application: "app1",
				From:        now.AddDate(0, 0, -3),
				To:          now.AddDate(0, 0, 3),
				Count: RelayCounts{
					Success: 3*2 + 50,
					Failure: 3*3 + 40,
				},
			},
		},
		{
			name: "Only today's metrics are included when the timespan only includes today",
			from: time.Now(),
			to:   time.Now().AddDate(0, 0, 2),
			expected: AppRelaysResponse{
				Application: "app1",
				From:        now,
				To:          now.AddDate(0, 0, 3),
				Count: RelayCounts{
					Success: 50,
					Failure: 40,
				},
			},
		},
		{
			name: "Missing parameters' default values",
			expected: AppRelaysResponse{
				Application: "app1",
				From:        now.AddDate(0, 0, -30),
				To:          now.AddDate(0, 0, 1),
				Count: RelayCounts{
					Success: 6*2 + 50,
					Failure: 6*3 + 40,
				},
			},
		},
		{
			name:        "Invalid timespan is rejected",
			from:        now.AddDate(0, 0, -1),
			to:          now.AddDate(0, 0, -2),
			expectedErr: fmt.Errorf("Invalid timespan"),
			expected: AppRelaysResponse{
				Application: "app1",
				From:        now.AddDate(0, 0, -1),
				To:          now.AddDate(0, 0, -2),
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fakeBackend := fakeBackend{
				usage:       usageData,
				todaysUsage: todaysUsage,
			}

			relayMeter := NewRelayMeter(&fakeBackend, logger.New(), RelayMeterOptions{LoadInterval: 100 * time.Millisecond})
			time.Sleep(200 * time.Millisecond)
			got, err := relayMeter.AppRelays(requestedApp, tc.from, tc.to)
			if err != nil {
				if tc.expectedErr == nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				if !strings.Contains(err.Error(), tc.expectedErr.Error()) {
					t.Fatalf("Expected error to contain: %q, got: %v", tc.expectedErr.Error(), err)
				}
			}

			if diff := cmp.Diff(tc.expected, got); diff != "" {
				t.Errorf("unexpected value (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAllAppsRelays(t *testing.T) {
	now, _ := time.Parse(dayFormat, time.Now().Format(dayFormat))
	usageData := fakeDailyMetrics()
	todaysUsage := fakeTodaysMetrics()

	testCases := []struct {
		name                string
		from                time.Time
		to                  time.Time
		usageData           map[time.Time]map[string]int64
		todaysUsage         map[string]int64
		expected            map[string]AppRelaysResponse
		expectedErr         error
		expectedTodaysCalls int
	}{
		{
			name: "Correct count is returned",
			from: now.AddDate(0, 0, -5),
			to:   now.AddDate(0, 0, -1),
			expected: map[string]AppRelaysResponse{
				"app1": {
					Application: "app1",
					From:        now.AddDate(0, 0, -5),
					To:          now,
					Count: RelayCounts{
						Success: 10,
						Failure: 15,
					},
				},
				"app2": {
					Application: "app2",
					From:        now.AddDate(0, 0, -5),
					To:          now,
					Count: RelayCounts{
						Success: 5,
						Failure: 25,
					},
				},
				"app4": {
					Application: "app4",
					From:        now.AddDate(0, 0, -5),
					To:          now,
					Count: RelayCounts{
						Success: 25,
						Failure: 35,
					},
				},
			},
		},
		{
			name: "From and To parameters are adjusted to start of the specifed day and the next day, respectively",
			from: time.Now().AddDate(0, 0, -5),
			to:   time.Now().AddDate(0, 0, -1),
			expected: map[string]AppRelaysResponse{
				"app1": {
					Application: "app1",
					From:        now.AddDate(0, 0, -5),
					To:          now,
					Count: RelayCounts{
						Success: 10,
						Failure: 15,
					},
				},
				"app2": {
					Application: "app2",
					From:        now.AddDate(0, 0, -5),
					To:          now,
					Count: RelayCounts{
						Success: 5,
						Failure: 25,
					},
				},
				"app4": {
					Application: "app4",
					From:        now.AddDate(0, 0, -5),
					To:          now,
					Count: RelayCounts{
						Success: 25,
						Failure: 35,
					},
				},
			},
		},
		{
			name: "Equal values are allowed for From and To parameters (to cover a single day)",
			from: now.AddDate(0, 0, -3),
			to:   now.AddDate(0, 0, -3),
			expected: map[string]AppRelaysResponse{
				"app1": {
					Application: "app1",
					From:        now.AddDate(0, 0, -3),
					To:          now.AddDate(0, 0, -2),
					Count: RelayCounts{
						Success: 2,
						Failure: 3,
					},
				},
				"app2": {
					Application: "app2",
					From:        now.AddDate(0, 0, -3),
					To:          now.AddDate(0, 0, -2),
					Count: RelayCounts{
						Success: 1,
						Failure: 5,
					},
				},
				"app4": {
					Application: "app4",
					From:        now.AddDate(0, 0, -3),
					To:          now.AddDate(0, 0, -2),
					Count: RelayCounts{
						Success: 5,
						Failure: 7,
					},
				},
			},
		},
		{
			name: "Today's metrics are added to previous days' usage data",
			from: now.AddDate(0, 0, -3),
			to:   now,
			todaysUsage: map[string]int64{
				"app1": 50,
				"app2": 30,
			},
			expected: map[string]AppRelaysResponse{
				"app1": {
					Application: "app1",
					From:        now.AddDate(0, 0, -3),
					To:          now.AddDate(0, 0, 1),
					Count: RelayCounts{
						Success: 56,
						Failure: 49,
					},
				},
				"app2": {
					Application: "app2",
					From:        now.AddDate(0, 0, -3),
					To:          now.AddDate(0, 0, 1),
					Count: RelayCounts{
						Success: 33,
						Failure: 85,
					},
				},
				"app4": {
					Application: "app4",
					From:        now.AddDate(0, 0, -3),
					To:          now.AddDate(0, 0, 1),
					Count: RelayCounts{
						Success: 515,
						Failure: 721,
					},
				},
			},
		},
		{
			name: "Only today's data is included when from and to parameters point to today",
			from: now,
			to:   now,
			expected: map[string]AppRelaysResponse{
				"app1": {
					Application: "app1",
					From:        now,
					To:          now.AddDate(0, 0, 1),
					Count: RelayCounts{
						Success: 50,
						Failure: 40,
					},
				},
				"app2": {
					Application: "app2",
					From:        now,
					To:          now.AddDate(0, 0, 1),
					Count: RelayCounts{
						Success: 30,
						Failure: 70,
					},
				},
				"app4": {
					Application: "app4",
					From:        now,
					To:          now.AddDate(0, 0, 1),
					Count: RelayCounts{
						Success: 500,
						Failure: 700,
					},
				},
			},
		},
		{
			name: "Today's metrics are not included when the 'to' parameter does not include today",
			from: time.Now().AddDate(0, 0, -3),
			to:   time.Now().AddDate(0, 0, -1),
			expected: map[string]AppRelaysResponse{
				"app1": {
					Application: "app1",
					From:        now.AddDate(0, 0, -3),
					To:          now,
					Count: RelayCounts{
						Success: 6,
						Failure: 9,
					},
				},
				"app2": {
					Application: "app2",
					From:        now.AddDate(0, 0, -3),
					To:          now,
					Count: RelayCounts{
						Success: 3,
						Failure: 15,
					},
				},
				"app4": {
					Application: "app4",
					From:        now.AddDate(0, 0, -3),
					To:          now,
					Count: RelayCounts{
						Success: 15,
						Failure: 21,
					},
				},
			},
		},
		{
			name: "Today's metrics are included when the 'to' parameter is after today",
			from: time.Now().AddDate(0, 0, -3),
			to:   time.Now().AddDate(0, 0, 2),
			expected: map[string]AppRelaysResponse{
				"app1": {
					Application: "app1",
					From:        now.AddDate(0, 0, -3),
					To:          now.AddDate(0, 0, 3),
					Count: RelayCounts{
						Success: 56,
						Failure: 49,
					},
				},
				"app2": {
					Application: "app2",
					From:        now.AddDate(0, 0, -3),
					To:          now.AddDate(0, 0, 3),
					Count: RelayCounts{
						Success: 33,
						Failure: 85,
					},
				},
				"app4": {
					Application: "app4",
					From:        now.AddDate(0, 0, -3),
					To:          now.AddDate(0, 0, 3),
					Count: RelayCounts{
						Success: 515,
						Failure: 721,
					},
				},
			},
		},
		{
			name: "Only today's metrics are included when the timespan only includes today",
			from: time.Now(),
			to:   time.Now().AddDate(0, 0, 2),
			expected: map[string]AppRelaysResponse{
				"app1": {
					Application: "app1",
					From:        now,
					To:          now.AddDate(0, 0, 3),
					Count: RelayCounts{
						Success: 50,
						Failure: 40,
					},
				},
				"app2": {
					Application: "app2",
					From:        now,
					To:          now.AddDate(0, 0, 3),
					Count: RelayCounts{
						Success: 30,
						Failure: 70,
					},
				},
				"app4": {
					Application: "app4",
					From:        now,
					To:          now.AddDate(0, 0, 3),
					Count: RelayCounts{
						Success: 500,
						Failure: 700,
					},
				},
			},
		},
		{
			name: "Missing parameters' default values",
			expected: map[string]AppRelaysResponse{
				"app1": {
					Application: "app1",
					From:        now.AddDate(0, 0, -30),
					To:          now.AddDate(0, 0, 1),
					Count: RelayCounts{
						Success: 62,
						Failure: 58,
					},
				},
				"app2": {
					Application: "app2",
					From:        now.AddDate(0, 0, -30),
					To:          now.AddDate(0, 0, 1),
					Count: RelayCounts{
						Success: 36,
						Failure: 100,
					},
				},
				"app4": {
					Application: "app4",
					From:        now.AddDate(0, 0, -30),
					To:          now.AddDate(0, 0, 1),
					Count: RelayCounts{
						Success: 530,
						Failure: 742,
					},
				},
			},
		},
		{
			name:        "Invalid timespan is rejected",
			from:        now.AddDate(0, 0, -1),
			to:          now.AddDate(0, 0, -2),
			expectedErr: fmt.Errorf("Invalid timespan"),
			expected:    map[string]AppRelaysResponse{},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fakeBackend := fakeBackend{
				usage:       usageData,
				todaysUsage: todaysUsage,
			}

			relayMeter := NewRelayMeter(&fakeBackend, logger.New(), RelayMeterOptions{LoadInterval: 100 * time.Millisecond})
			time.Sleep(200 * time.Millisecond)
			rawGot, err := relayMeter.AllAppsRelays(tc.from, tc.to)
			if err != nil {
				if tc.expectedErr == nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				if !strings.Contains(err.Error(), tc.expectedErr.Error()) {
					t.Fatalf("Expected error to contain: %q, got: %v", tc.expectedErr.Error(), err)
				}
			}

			// Need to convert it to map to be able to compare
			got := make(map[string]AppRelaysResponse, len(rawGot))

			for _, relResp := range rawGot {
				got[relResp.Application] = relResp
			}

			if diff := cmp.Diff(tc.expected, got); diff != "" {
				t.Errorf("unexpected value (-want +got):\n%s", diff)
			}
		})
	}
}

func TestLoadBalancerRelays(t *testing.T) {
	now, _ := time.Parse(dayFormat, time.Now().Format(dayFormat))
	usageData := fakeDailyMetrics()
	todaysUsage := fakeTodaysMetrics()
	errBackendFailure := errors.New("backend error")

	testCases := []struct {
		name         string
		loadbalancer string
		from         time.Time
		to           time.Time
		backendErr   error

		expected    LoadBalancerRelaysResponse
		expectedErr error
	}{
		{
			name:        "LoadBalancer not found error",
			expectedErr: ErrLoadBalancerNotFound,
		},
		{
			name:        "Backend service error",
			backendErr:  errBackendFailure,
			expectedErr: errBackendFailure,
		},
		{
			name:         "Correct summary for a loadbalancer",
			loadbalancer: "lb1",
			from:         now.AddDate(0, 0, -6),
			to:           now,
			expected: LoadBalancerRelaysResponse{
				From:         now.AddDate(0, 0, -6),
				To:           now.AddDate(0, 0, 1),
				Endpoint:     "lb1",
				Applications: []string{"app1", "app2", "app3"},
				Count: RelayCounts{
					Success: 6*(2+1) + 50 + 30,
					Failure: 6*(3+5) + 40 + 70,
				},
			},
		},
		{
			name:         "Correct summary for a loadbalancer excluding today",
			loadbalancer: "lb1",
			from:         now.AddDate(0, 0, -6),
			to:           now.AddDate(0, 0, -2),
			expected: LoadBalancerRelaysResponse{
				From:         now.AddDate(0, 0, -6),
				To:           now.AddDate(0, 0, -1),
				Endpoint:     "lb1",
				Applications: []string{"app1", "app2", "app3"},
				Count: RelayCounts{
					Success: 5 * (2 + 1),
					Failure: 5 * (3 + 5),
				},
			},
		},
		{
			name:         "Correct summary for a loadbalancer on todays metrics",
			loadbalancer: "lb1",
			from:         now,
			to:           now,
			expected: LoadBalancerRelaysResponse{
				From:         now,
				To:           now.AddDate(0, 0, 1),
				Endpoint:     "lb1",
				Applications: []string{"app1", "app2", "app3"},
				Count: RelayCounts{
					Success: 50 + 30,
					Failure: 40 + 70,
				},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fakeBackend := fakeBackend{
				usage:       usageData,
				todaysUsage: todaysUsage,
				loadbalancers: map[string]*repository.LoadBalancer{
					"lb1": {
						Applications: []*repository.Application{
							{GatewayAAT: repository.GatewayAAT{ApplicationPublicKey: "app1"}},
							{GatewayAAT: repository.GatewayAAT{ApplicationPublicKey: "app2"}},
							{GatewayAAT: repository.GatewayAAT{ApplicationPublicKey: "app3"}},
						},
					},
					"lb2": {
						Applications: []*repository.Application{
							{GatewayAAT: repository.GatewayAAT{ApplicationPublicKey: "app4"}},
							{GatewayAAT: repository.GatewayAAT{ApplicationPublicKey: "app5"}},
							{GatewayAAT: repository.GatewayAAT{ApplicationPublicKey: "app6"}},
						},
					},
				},
				err: tc.backendErr,
			}

			relayMeter := NewRelayMeter(&fakeBackend, logger.New(), RelayMeterOptions{LoadInterval: 100 * time.Millisecond})
			time.Sleep(200 * time.Millisecond)
			got, err := relayMeter.LoadBalancerRelays(tc.loadbalancer, tc.from, tc.to)
			if err != nil && !errors.Is(err, tc.expectedErr) {
				t.Fatalf("Expected error: %v, got: %v", tc.expectedErr, err)
			}

			if diff := cmp.Diff(tc.expected, got); diff != "" {
				t.Errorf("unexpected value (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAllLoadBalancersRelays(t *testing.T) {
	now, _ := time.Parse(dayFormat, time.Now().Format(dayFormat))
	usageData := fakeDailyMetrics()
	todaysUsage := fakeTodaysMetrics()
	errBackendFailure := errors.New("backend error")

	testCases := []struct {
		name       string
		from       time.Time
		to         time.Time
		backendErr error

		expected    map[string]LoadBalancerRelaysResponse
		expectedErr error
	}{
		{
			name:        "Backend service error",
			backendErr:  errBackendFailure,
			expectedErr: errBackendFailure,
			expected:    map[string]LoadBalancerRelaysResponse{},
		},
		{
			name: "Correct summary for loadbalancers",
			from: now.AddDate(0, 0, -6),
			to:   now,
			expected: map[string]LoadBalancerRelaysResponse{
				"lb1": {
					From:         now.AddDate(0, 0, -6),
					To:           now.AddDate(0, 0, 1),
					Endpoint:     "lb1",
					Applications: []string{"app1", "app2", "app3"},
					Count: RelayCounts{
						Success: 98,
						Failure: 158,
					},
				},
				"lb2": {
					From:         now.AddDate(0, 0, -6),
					To:           now.AddDate(0, 0, 1),
					Endpoint:     "lb2",
					Applications: []string{"app4", "app5", "app6"},
					Count: RelayCounts{
						Success: 530,
						Failure: 742,
					},
				},
			},
		},
		{
			name: "Correct summary for loadbalancers excluding today",
			from: now.AddDate(0, 0, -6),
			to:   now.AddDate(0, 0, -2),
			expected: map[string]LoadBalancerRelaysResponse{
				"lb1": {
					From:         now.AddDate(0, 0, -6),
					To:           now.AddDate(0, 0, -1),
					Endpoint:     "lb1",
					Applications: []string{"app1", "app2", "app3"},
					Count: RelayCounts{
						Success: 15,
						Failure: 40,
					},
				},
				"lb2": {
					From:         now.AddDate(0, 0, -6),
					To:           now.AddDate(0, 0, -1),
					Endpoint:     "lb2",
					Applications: []string{"app4", "app5", "app6"},
					Count: RelayCounts{
						Success: 25,
						Failure: 35,
					},
				},
			},
		},
		{
			name: "Correct summary for loadbalancers on todays metrics",
			from: now,
			to:   now,
			expected: map[string]LoadBalancerRelaysResponse{
				"lb1": {
					From:         now,
					To:           now.AddDate(0, 0, 1),
					Endpoint:     "lb1",
					Applications: []string{"app1", "app2", "app3"},
					Count: RelayCounts{
						Success: 80,
						Failure: 110,
					},
				},
				"lb2": {
					From:         now,
					To:           now.AddDate(0, 0, 1),
					Endpoint:     "lb2",
					Applications: []string{"app4", "app5", "app6"},
					Count: RelayCounts{
						Success: 500,
						Failure: 700,
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fakeBackend := fakeBackend{
				usage:       usageData,
				todaysUsage: todaysUsage,
				loadbalancers: map[string]*repository.LoadBalancer{
					"lb1": {
						ID: "lb1",
						Applications: []*repository.Application{
							{GatewayAAT: repository.GatewayAAT{ApplicationPublicKey: "app1"}},
							{GatewayAAT: repository.GatewayAAT{ApplicationPublicKey: "app2"}},
							{GatewayAAT: repository.GatewayAAT{ApplicationPublicKey: "app3"}},
						},
					},
					"lb2": {
						ID: "lb2",
						Applications: []*repository.Application{
							{GatewayAAT: repository.GatewayAAT{ApplicationPublicKey: "app4"}},
							{GatewayAAT: repository.GatewayAAT{ApplicationPublicKey: "app5"}},
							{GatewayAAT: repository.GatewayAAT{ApplicationPublicKey: "app6"}},
						},
					},
				},
				err: tc.backendErr,
			}

			relayMeter := NewRelayMeter(&fakeBackend, logger.New(), RelayMeterOptions{LoadInterval: 100 * time.Millisecond})
			time.Sleep(200 * time.Millisecond)
			rawGot, err := relayMeter.AllLoadBalancersRelays(tc.from, tc.to)
			if err != nil && !errors.Is(err, tc.expectedErr) {
				t.Fatalf("Expected error: %v, got: %v", tc.expectedErr, err)
			}

			// Need to convert it to map to be able to compare
			got := make(map[string]LoadBalancerRelaysResponse, len(rawGot))

			for _, relResp := range rawGot {
				got[relResp.Endpoint] = relResp
			}

			if diff := cmp.Diff(tc.expected, got); diff != "" {
				t.Errorf("unexpected value (-want +got):\n%s", diff)
			}
		})
	}
}

func TestStartDataLoader(t *testing.T) {
	now, _ := time.Parse(dayFormat, time.Now().Format(dayFormat))

	testCases := []struct {
		name          string
		maxArchiveAge time.Duration
		expectedFrom  time.Time
		expectedTo    time.Time
	}{
		{
			name:         "Default maxArchiveAge is applied",
			expectedFrom: now.AddDate(0, 0, -1*MAX_PAST_DAYS_METRICS_DEFAULT_DAYS),
			expectedTo:   now,
		},
		{
			name:          "The specified maxArchiveAge takes effect",
			maxArchiveAge: 24 * 5 * time.Hour,
			expectedFrom:  now.AddDate(0, 0, -5),
			expectedTo:    now,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fakeBackend := fakeBackend{}
			meter := &relayMeter{
				Backend: &fakeBackend,
				Logger:  logger.New(),
				RelayMeterOptions: RelayMeterOptions{
					LoadInterval: 1 * time.Second,
					MaxPastDays:  tc.maxArchiveAge,
				},
			}

			ctx, cancel := context.WithCancel(context.Background())
			go func() {
				meter.StartDataLoader(ctx)
			}()
			// We only need the initial data loader run
			time.Sleep(10 * time.Millisecond)
			cancel()

			if !tc.expectedFrom.Equal(fakeBackend.dailyMetricsFrom) {
				t.Errorf("Expected 'from' to be: %v, got: %v", tc.expectedFrom, fakeBackend.dailyMetricsFrom)
			}

		})
	}
}

type fakeBackend struct {
	usage       map[time.Time]map[string]RelayCounts
	err         error
	todaysUsage map[string]RelayCounts
	userApps    map[string][]string

	todaysMetricsCalls int
	dailyMetricsCalls  int
	dailyMetricsFrom   time.Time
	dailyMetricsTo     time.Time

	loadbalancers map[string]*repository.LoadBalancer
}

func (f *fakeBackend) DailyUsage(from, to time.Time) (map[time.Time]map[string]RelayCounts, error) {
	f.dailyMetricsCalls++
	f.dailyMetricsFrom = from
	f.dailyMetricsTo = to
	return f.usage, f.err
}

func (f *fakeBackend) TodaysUsage() (map[string]RelayCounts, error) {
	f.todaysMetricsCalls++
	return f.todaysUsage, nil
}

func (f *fakeBackend) UserApps(user string) ([]string, error) {
	return f.userApps[user], nil
}

func (f *fakeBackend) LoadBalancer(endpoint string) (*repository.LoadBalancer, error) {
	return f.loadbalancers[endpoint], f.err
}

func (f *fakeBackend) LoadBalancers() ([]*repository.LoadBalancer, error) {
	var lbs []*repository.LoadBalancer

	for _, lb := range f.loadbalancers {
		lbs = append(lbs, lb)
	}

	return lbs, f.err
}

func fakeDailyMetrics() map[time.Time]map[string]RelayCounts {
	dayMetrics := map[string]RelayCounts{
		"app1": {Success: 2, Failure: 3},
		"app2": {Success: 1, Failure: 5},
		"app4": {Success: 5, Failure: 7},
	}

	now, _ := time.Parse(dayFormat, time.Now().Format(dayFormat))
	metrics := make(map[time.Time]map[string]RelayCounts)
	for i := 1; i < 7; i++ {
		metrics[now.AddDate(0, 0, -1*i)] = dayMetrics
	}
	return metrics
}

func fakeTodaysMetrics() map[string]RelayCounts {
	return map[string]RelayCounts{
		"app1": {Success: 50, Failure: 40},
		"app2": {Success: 30, Failure: 70},
		"app4": {Success: 500, Failure: 700},
	}
}
