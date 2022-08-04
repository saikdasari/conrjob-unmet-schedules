package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/robfig/cron"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	dat, err := os.ReadFile("job.json")
	check(err)

	var cj batchv1beta1.CronJob
	err = json.Unmarshal(dat, &cj)
	check(err)
	fmt.Println("")
	fmt.Printf("********* Last Schedule Time: %s *********\n", cj.Status.LastScheduleTime.String())

	// PDT
	loc, e := time.LoadLocation("America/Los_Angeles")
	check(e)

	type TestCase struct {
		description string
		time        time.Time
	}

	test_cases := []TestCase{
		{"exactly on schedule", time.Date(2022, 7, 21, 3, 0, 0, 0, loc)},
		{"after the schedule, before startingDeadlineSeconds", time.Date(2022, 7, 21, 3, 0, 9, 0, loc)},
		{"exactly on startDeadlineSeconds", time.Date(2022, 7, 21, 3, 0, 10, 0, loc)},
		{"after the schedule", time.Date(2022, 7, 21, 3, 0, 11, 0, loc)},
	}

	for _, tc := range test_cases {
		fmt.Printf("\nCronJob controller running %s at %s\n", tc.description, tc.time.String())
		fmt.Println("Unmet Schedules:")
		getRecentUnmetScheduleTimes(cj, tc.time)
	}

	fmt.Println("\n\n****** END ******")
}

func getRecentUnmetScheduleTimes(cj batchv1beta1.CronJob, now time.Time) ([]time.Time, error) {
	starts := []time.Time{}
	sched, err := cron.ParseStandard(cj.Spec.Schedule) // This parser uses machine timezone
	if err != nil {
		return starts, fmt.Errorf("unparseable schedule: %s : %s", cj.Spec.Schedule, err)
	}

	var earliestTime time.Time
	if cj.Status.LastScheduleTime != nil {
		earliestTime = cj.Status.LastScheduleTime.Time
	} else {
		// If none found, then this is either a recently created cronJob,
		// or the active/completed info was somehow lost (contract for status
		// in kubernetes says it may need to be recreated), or that we have
		// started a job, but have not noticed it yet (distributed systems can
		// have arbitrary delays).  In any case, use the creation time of the
		// CronJob as last known start time.
		earliestTime = cj.ObjectMeta.CreationTimestamp.Time
	}
	if cj.Spec.StartingDeadlineSeconds != nil {
		// Controller is not going to schedule anything below this point
		schedulingDeadline := now.Add(-time.Second * time.Duration(*cj.Spec.StartingDeadlineSeconds))

		if schedulingDeadline.After(earliestTime) {
			earliestTime = schedulingDeadline
		}
	}
	if earliestTime.After(now) {
		return []time.Time{}, nil
	}

	for t := sched.Next(earliestTime); !t.After(now); t = sched.Next(t) {
		fmt.Printf("	- %s\n", t.String())
		starts = append(starts, t)
		// An object might miss several starts. For example, if
		// controller gets wedged on friday at 5:01pm when everyone has
		// gone home, and someone comes in on tuesday AM and discovers
		// the problem and restarts the controller, then all the hourly
		// jobs, more than 80 of them for one hourly cronJob, should
		// all start running with no further intervention (if the cronJob
		// allows concurrency and late starts).
		//
		// However, if there is a bug somewhere, or incorrect clock
		// on controller's server or apiservers (for setting creationTimestamp)
		// then there could be so many missed start times (it could be off
		// by decades or more), that it would eat up all the CPU and memory
		// of this controller. In that case, we want to not try to list
		// all the missed start times.
		//
		// I've somewhat arbitrarily picked 100, as more than 80,
		// but less than "lots".
		if len(starts) > 100 {
			// We can't get the most recent times so just return an empty slice
			return []time.Time{}, fmt.Errorf("too many missed start time (> 100). Set or decrease .spec.startingDeadlineSeconds or check clock skew")
		}
	}
	return starts, nil
}
