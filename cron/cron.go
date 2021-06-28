package cron

import (
	"os"
	"fmt"
	"bufio"
	"math"
	"os/exec"
	"time"
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

func New(rpc_url string) (*Cron, error) {
	c := &Cron{
		parser:    standardParser,
		schedules:   nil,
	}

	err := c.Init(rpc_url)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Cron) Init(rpc_url string) (error) {
	fmt.Println("INIT", rpc_url)
	
	api, err := NewSubstrateUtils(rpc_url)
	if err != nil {
		return err
	}

	c.api = api
	
	return nil
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

func (c *Cron) RunJobs(LocalSlot, LocalSession, CurrentEra, BlockNumber uint64) {
	for _,schedule := range c.schedules {
		if (matchTrigger(LocalSlot, schedule.Slot) && matchTrigger(LocalSession, schedule.Session) && matchTrigger(CurrentEra, schedule.Era)) {
			cmd := exec.Command(schedule.Job.Exec, schedule.Job.Args...)
			cmd.Stdout = os.Stdout
			cmd.Env = append(os.Environ(),
				fmt.Sprintf("LOCAL_SLOT=%d", LocalSlot),
				fmt.Sprintf("LOCAL_SESSION=%d", LocalSession),
				fmt.Sprintf("CURRENT_ERA=%d", CurrentEra),
				fmt.Sprintf("BLOCK_NUMBER=%d", BlockNumber),
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
	fmt.Println("RUN!")

	sub_heads, err := c.api.RPC.Chain.SubscribeNewHeads()
	if err != nil {
		return err
	}

	LatestBlockNumber := uint64(0)
	timeoutCount := 0

	fmt.Println("LOOP")
	for {
		if timeoutCount > 20 {
			err = c.Init(c.api.Client.URL())
			if err == nil {
				sub_heads, err = c.api.RPC.Chain.SubscribeNewHeads()
				if err != nil {
					return err
				}

				timeoutCount = 0
			} else {
				time.Sleep(5 * time.Second)
				continue
			}
		}
		select {
			case head := <-sub_heads.Chan():
				timeoutCount = 0
				if (uint64(head.Number) > LatestBlockNumber) {
					block, err := c.api.RPC.Chain.GetBlockLatest()
					if err != nil {
						break
					}
					LatestBlockNumber = uint64(block.Block.Header.Number)

					info, err := c.api.GetSessionInfo()
					if err != nil {
						break
					}

					c.RunJobs(info.GetLocalSlot(),  info.GetLocalSession(),  info.GetCurrentEra(), LatestBlockNumber)
				}
			case err := <-sub_heads.Err():
				fmt.Println("ERROR", err)
			case <-time.After(1 * time.Second):
				fmt.Println("TIMEOUT", timeoutCount)
				timeoutCount += 1	
		}		
	} 

	return err		
} 
/*
func (c *Cron) connectAndSubscribe(url string) (error {
	api, err = NewSubstrateUtils(url)
	c.api = api

}
*/

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
