package cron

import (
	"fmt"
	"strconv"
	"strings"
	"regexp"
)

import shellquote "github.com/kballard/go-shellquote"

type ParseOption int

const (
	Slot         ParseOption = 1 << iota // Slot trigger, default 0
	SlotOptional                         // Optional slot trigger, default 0
	Session                              // Epoch trigger, default 0
	Era                                  // Era trigger, default 0
)

var places = []ParseOption{
	Slot,
	Session,
	Era,
}

var defaults = []string{
	"*",
	"*",
	"*",
}


type Parser struct {
	options ParseOption
}

func NewParser(options ParseOption) Parser {
	optionals := 0
	if options&SlotOptional > 0 {
		optionals++
	}
	if optionals > 1 {
		panic("multiple optionals may not be configured")
	}
	return Parser{options}
}

func (p Parser) Parse(spec string) (*Schedule, error) {
	if len(spec) == 0 {
		return nil, fmt.Errorf("empty spec string")
	}

    fields,err := shellquote.Split(spec)
    if err != nil {
    	return nil,err
    }

	var idx int
	IsField := regexp.MustCompile(`^*[0-9/,\-\*]+$`).MatchString
	for i,f := range fields {
		if !IsField(f) {
			idx = i
			break
		}
	}

	triggers, err := normalizeTriggers(fields[:idx], p.options)
	if err != nil {
		return nil, err
	}

	exec := fields[idx]
	args := fields[idx+1:]

	slot, err := p.ParseTrigger(triggers[0])
	if err != nil {
		return nil, err
	}
	session, err := p.ParseTrigger(triggers[1])
	if err != nil {
		return nil, err
	}
	era, err := p.ParseTrigger(triggers[2])
	if err != nil {
		return nil, err
	}

	return &Schedule{
		Slot:  slot,
		Session: session,
		Era:   era,
		Job: &ScheduleJob{
			Exec: exec,
			Args: args,
		},
	}, nil
}

func (p Parser) ParseTrigger(expr string) (*ScheduleTrigger, error) {
	var err error

	st := &ScheduleTrigger{}

	SplitStep := strings.Split(expr, "/")
	if len(SplitStep) > 2 {
		return nil, fmt.Errorf("bad step format - to many '/'")
	}
	if len(SplitStep) == 1 {
		st.Step = 1
	} else {
		st.Step, err = strconv.ParseUint(SplitStep[1], 10, 64)
		if err != nil { 
			return nil, fmt.Errorf("bad step format - not a number")
		}
	}

	SplitItems := strings.Split(SplitStep[0], ",")
	if len(SplitItems) == 1 {
		if SplitItems[0] == "*" {
			st.Any = true
		}
	}
	if !st.Any { 
		for _,Item :=range SplitItems {
			SplitRange := strings.Split(Item, "-")

			if len(SplitRange) > 2 {
				return nil, fmt.Errorf("bad range format - to many '-'")
			}

			NewRange := TriggerRange{}

			NewRange.Lower, err = strconv.ParseUint(SplitRange[0], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("bad range format - not a number")
			}			
			if len(SplitRange) == 1 {
				NewRange.Higher = NewRange.Lower
			} else {
				NewRange.Higher, err = strconv.ParseUint(SplitRange[1], 10, 64)
				if err != nil {
					return nil, fmt.Errorf("bad range format - not a number")
				}	
			}

			if NewRange.Lower > NewRange.Higher {
				return nil, fmt.Errorf("bad range format - decreasing range")
			}

			st.Ranges = append(st.Ranges, NewRange)
		}
	}

	return st, nil
}

func normalizeTriggers(fields []string, options ParseOption) ([]string, error) {
	optionals := 0
	if options&SlotOptional > 0 {
		options |= Slot
		optionals++
	}
	if optionals > 1 {
		return nil, fmt.Errorf("multiple optionals may not be configured")
	}

	max := 0
	for _, place := range places {
		if options&place > 0 {
			max++
		}
	}
	min := max - optionals

	if count := len(fields); count < min || count > max {
		if min == max {
			return nil, fmt.Errorf("expected exactly %d fields, found %d: %s", min, count, fields)
		}
		return nil, fmt.Errorf("expected %d to %d fields, found %d: %s", min, max, count, fields)
	}

	if min < max && len(fields) == min {
		switch {
		case options&SlotOptional > 0:
			fields = append([]string{defaults[0]}, fields...)
		default:
			return nil, fmt.Errorf("unknown optional field")
		}
	}

	n := 0
	expandedFields := make([]string, len(places))
	copy(expandedFields, defaults)
	for i, place := range places {
		if options&place > 0 {
			expandedFields[i] = fields[n]
			n++
		}
	}

	return expandedFields, nil
}

var standardParser = NewParser(
	Slot | Session | Era,
)