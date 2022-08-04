# cronjob-unmet-schedules
The cronjob-controller [evaluates](https://github.com/kubernetes/kubernetes/blob/v1.20.15/pkg/controller/cronjob/cronjob_controller.go#L267-L298) the unmet schedules (times the job should have started but did not). 
LastScheduledTime, startingDeadlineSeconds are couple of variables that affect the unmet schedules. 
If there are no unmet schedules, the job will not be scheduled. 

This repo helps in understanding what the unmet schedules are in different scenarios.

The tests assert that a cronjob will only have unmet schedules if
```scheduled time <= controller time < (scheduled time + startingDeadlineSeconds)```

## Instructions
In the `job.json`, spec.schedule is set to `0 3 * * *` to reflect 3 A.M. in Pacific Time Zone. 
If you are not in Pacific Time Zone, update this value. 

For example, update this to `0 5 * * *` when running this from Central Time Zone.