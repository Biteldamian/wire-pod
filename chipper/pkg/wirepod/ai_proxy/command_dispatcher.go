package ai_proxy

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/fforchino/vector-go-sdk/pkg/vector"
	"github.com/fforchino/vector-go-sdk/pkg/vectorpb"
	"github.com/kercre123/wire-pod/chipper/pkg/logger"
	"github.com/kercre123/wire-pod/chipper/pkg/vars"
)

type AICommand struct {
	Type string `json:"type"` // e.g., "move", "animate"
	Param string `json:"param"` // e.g., "forward", "happy"
}

func DispatchCommand(robot *vector.Vector, cmdJson string) error {
	var cmd AICommand
	if err := json.Unmarshal([]byte(cmdJson), &cmd); err != nil {
		return err
	}

	switch cmd.Type {
	case "say":
		robot.Conn.SayText(context.Background(), &vectorpb.SayTextRequest{Text: cmd.Param, UseVectorVoice: true})
	case "move":
		if cmd.Param == "forward" {
			robot.Conn.DriveStraight(context.Background(), &vectorpb.DriveStraightRequest{DistMm: 100, SpeedMmps: 50})
		} // Add more params
	case "animate":
		robot.Conn.PlayAnimation(context.Background(), &vectorpb.PlayAnimationRequest{Animation: &vectorpb.Animation{Name: cmd.Param}, Loops: 1})
	default:
		return errors.New("unknown command: " + cmd.Type)
	}
	logger.Println("Dispatched command: " + cmd.Type)
	return nil
}