package tracker

import (
	"context"
	"fmt"
	"log"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	tellorCommon "github.com/tellor-io/TellorMiner/common"
	"github.com/tellor-io/TellorMiner/config"
	"github.com/tellor-io/TellorMiner/contracts"
	"github.com/tellor-io/TellorMiner/db"
	"github.com/tellor-io/TellorMiner/util"
)

var currentVarsLog = util.NewLogger("tracker", "CurrentVarsTracker")

//CurrentVariablesTracker concrete tracker type
type CurrentVariablesTracker struct {
}

func (b *CurrentVariablesTracker) String() string {
	return "CurrentVariablesTracker"
}

//Exec implementation for tracker
func (b *CurrentVariablesTracker) Exec(ctx context.Context) error {
	//cast client using type assertion since context holds generic interface{}
	DB := ctx.Value(tellorCommon.DBContextKey).(db.DB)
	//get the single config instance
	cfg, err := config.GetConfig()
	if err != nil {
		log.Fatal(err)
		return err
	}

	//get address from config
	_fromAddress := cfg.PublicAddress

	//convert to address
	fromAddress := common.HexToAddress(_fromAddress)

	instance := ctx.Value(tellorCommon.MasterContractContextKey).(*contracts.TellorMaster)
	currentChallenge, requestID, difficulty, queryString, granularity, totalTip, err := instance.GetCurrentVariables(nil)
	if err != nil {
		fmt.Println("Current Variables Retrieval Error")
		return err
	}

	pendingChallenge, err := DB.Get(db.PendingChallengeKey)
	if err != nil {
		fmt.Println("Problem retrieving pending challenge", err)
		return err
	}
	if len(pendingChallenge) > 0 {
		var asBytes32 [32]byte
		copy(asBytes32[:], pendingChallenge)
		didMine, err := instance.DidMine(nil, asBytes32, fromAddress)
		if err != nil {
			fmt.Println("Problem checking pending txn for mining status", err)
			return err
		}
		if didMine {
			fmt.Println("Removing now-confirmed pending challenge from DB")
			err = DB.Delete(db.PendingChallengeKey)
			if err != nil {
				fmt.Println("Could not delete pending challenge txn", err)
				return err
			}
		}
	}

	//if we've mined it, don't save it
	myStatus, err := instance.DidMine(nil, currentChallenge, fromAddress)
	if err != nil {
		fmt.Println("My Status Retrieval Error")
		return err
	}
	bitSetVar := []byte{0}
	if myStatus {
		bitSetVar = []byte{1}
	}
	currentVarsLog.Info("Retrieved variables. challengeHash: %v", currentChallenge)

	err = DB.Put(db.CurrentChallengeKey, currentChallenge[:])
	if err != nil {
		fmt.Println("Current Variables Put Error")
		return err
	}
	err = DB.Put(db.RequestIdKey, []byte(hexutil.EncodeBig(requestID)))
	if err != nil {
		fmt.Println("Current Variables Put Error")
		return err
	}
	err = DB.Put(db.DifficultyKey, []byte(hexutil.EncodeBig(difficulty)))
	if err != nil {
		fmt.Println("Current Variables Put Error")
		return err
	}
	err = DB.Put(db.QueryStringKey, []byte(queryString))
	if err != nil {
		fmt.Println("Current Variables Put Error")
		return err
	}
	err = DB.Put(db.GranularityKey, []byte(hexutil.EncodeBig(granularity)))
	if err != nil {
		fmt.Println("Current Variables Put Error")
		return err
	}
	err = DB.Put(db.TotalTipKey, []byte(hexutil.EncodeBig(totalTip)))
	if err != nil {
		fmt.Println("Current Variables Put Error")
		return err
	}

	return DB.Put(db.MiningStatusKey, bitSetVar)
}
