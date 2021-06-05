package cron

import (
	"os"
	"fmt"
	"bufio"
	"math"
	"os/exec"

	"github.com/PolkaFoundry/go-substrate-rpc-client/v3/types"
)

type Cron struct {
	parser    Parser
	schedules   []*Schedule
	api *SubstrateUtils
}

type TriggerRange struct {
	Lower, Higher uint64
}

type ScheduleTrigger struct {
	Any bool

	Ranges []TriggerRange

	Step	uint64
}

type ScheduleJob struct {
	Exec string
	Args []string
}

type Schedule struct {
	Slot *ScheduleTrigger
	Session *ScheduleTrigger
	Era *ScheduleTrigger
	Job *ScheduleJob
}

func New(RPC_URL string) (*Cron, error) {
	api, err := NewSubstrateUtils(RPC_URL)
	if err != nil {
		return nil, err
	}

	c := &Cron{
		parser:    standardParser,
		schedules:   nil,
		api: api,
	}
	return c, nil
}

func (c *Cron) LoadCrontab(filename string) error {
    fh, err := os.Open(filename)
    if err != nil {
        return err
    }
    defer fh.Close()

    scanner := bufio.NewScanner(fh)
    for scanner.Scan() {
        err := c.LoadExpression(scanner.Text())
        if err != nil {
        	return err
        }
    }

    if err := scanner.Err(); err != nil {
        return err
    }

    return nil
}

func (c *Cron) LoadExpression(expr string) (error) {
	schedule, err := c.parser.Parse(expr)
	if err != nil {
		return err
	}
	
	c.LoadSchedule(schedule)

	return nil
}

func (c *Cron) LoadSchedule(schedule *Schedule) {
	c.schedules = append(c.schedules, schedule)
}

func (c *Cron) RunJobs(LocalSlot, LocalSession, CurrentEra uint64) {
	for _,schedule := range c.schedules {
		if (matchTrigger(LocalSlot, schedule.Slot) && matchTrigger(LocalSession, schedule.Session) && matchTrigger(CurrentEra, schedule.Era)) {
			cmd := exec.Command(schedule.Job.Exec, schedule.Job.Args...)
			cmd.Stdout = os.Stdout
			cmd.Env = append(os.Environ(),
				fmt.Sprintf("LOCAL_SLOT=%d", LocalSlot),
				fmt.Sprintf("LOCAL_SESSION=%d", LocalSession),
				fmt.Sprintf("CURRENT_ERA=%d", CurrentEra),
			)
			go cmd.Run()
		}
	}
}

func (c *Cron) RunJob(schedule *Schedule) {
	cmd := exec.Command(schedule.Job.Exec, schedule.Job.Args...)
	cmd.Stdout = os.Stdout
	cmd.Env = append(os.Environ(),
		"LOCAL_SLOT=duplicate_value",
		"LOCAL_SESSION=actual_value",
		"CURRENT_ERA=",
	)
	go cmd.Run()
}	

func (c *Cron) Run() (error) {
	fmt.Println("Run!")

	info, err := c.api.GetSessionInfo()
	if err != nil {
		return err
	}

	key, err := types.CreateStorageKey(c.api.meta, "System", "Events", nil, nil)
	if err != nil {
		return err
	}

	sub_events, err := c.api.RPC.State.SubscribeStorageRaw([]types.StorageKey{key})
	if err != nil {
		return err
	}
	defer sub_events.Unsubscribe()	

	sub_heads, err := c.api.RPC.Chain.SubscribeNewHeads()
	if err != nil {
		return err
	}
	defer sub_heads.Unsubscribe()

	LatestBlockNumber := uint64(0)
	for {
		head := <-sub_heads.Chan()
		if !(uint64(head.Number) > LatestBlockNumber) {
			continue
		}
		LatestBlockNumber = uint64(head.Number)

		set := <-sub_events.Chan()

		for _, chng := range set.Changes {
			if !types.Eq(chng.StorageKey, key) || !chng.HasStorageData {
				continue
			}

			events := types.EventRecords{}
			types.EventRecordsRaw(chng.StorageData).DecodeEventRecords(c.api.meta, &events)
			if err != nil {
				continue
			}
		
			if len(events.Session_NewSession) > 0 {
				info, err = c.api.GetSessionInfo()
				if err != nil {
					return err
				}					
			}
		}

		if (uint64(head.Number) > info.CurrentStart + info.Config.Duration) {
			info, err = c.api.GetSessionInfo()
			if err != nil {
				return err
			}			
		}
		LocalSlot := uint64(head.Number) - info.CurrentStart
		LocalSession := math.Mod(float64(info.CurrentIndex), float64(info.Config.SessionsPerEra))

		SessionPercentage := float64(LocalSlot)/float64(info.Config.Duration)
		EraPercentage := (LocalSession * float64(info.Config.Duration) + float64(LocalSlot))/(float64(info.Config.SessionsPerEra)*(float64(info.Config.Duration)))

		_ = SessionPercentage
		_ = EraPercentage

		c.RunJobs(LocalSlot, uint64(LocalSession), uint64(info.CurrentEra))
	}

}


func matchTrigger(value uint64, trigger *ScheduleTrigger) (bool) {
	match := false
	if trigger.Any {
		match = true
	} else {
		for _,tr := range trigger.Ranges {
			if (value >= tr.Lower && value <= tr.Higher) {
				match = true
				break
			}
		}

	}
	
	if !match {
		return false
	}

	return (math.Mod(float64(value), float64(trigger.Step)) == 0)
}