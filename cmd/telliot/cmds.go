// Copyright (c) The Tellor Authors.
// Licensed under the MIT License.

package main

import (
	"context"
	"fmt"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/go-kit/kit/log/level"
	"github.com/oklog/run"
	"github.com/pkg/errors"
	"github.com/prometheus/prometheus/tsdb"
	"github.com/tellor-io/telliot/pkg/aggregator"
	"github.com/tellor-io/telliot/pkg/db"
	"github.com/tellor-io/telliot/pkg/ethereum"
	"github.com/tellor-io/telliot/pkg/logging"
	"github.com/tellor-io/telliot/pkg/ops"
	"github.com/tellor-io/telliot/pkg/tracker/index"
	"github.com/tellor-io/telliot/pkg/tracker/profit"
	"github.com/tellor-io/telliot/pkg/web"
)

var GitTag string
var GitHash string

const versionMessage = `
    The official Tellor cli tool %s (%s)
    -----------------------------------------
	Website: https://tellor.io
	Github:  https://github.com/tellor-io/telliot
`

type VersionCmd struct {
}

func (cmd *VersionCmd) Run() error {
	//lint:ignore faillint it should print to console
	fmt.Printf(versionMessage, GitTag, GitHash)
	return nil
}

type configPath string
type tokenCmd struct {
	Config  configPath `type:"existingfile" help:"path to config file"`
	Address string     `arg:""`
	Amount  string     `arg:""`
	Account int        `arg:"" optional:""`
}

type transferCmd tokenCmd

func (c *transferCmd) Run() error {
	cfg, err := parseConfig(string(c.Config))
	if err != nil {
		return errors.Wrapf(err, "creating config")
	}

	logger := logging.NewLogger()

	ctx := context.Background()
	client, contract, accounts, err := createTellorVariables(ctx, logger, cfg.Ethereum)
	if err != nil {
		return errors.Wrapf(err, "creating tellor variables")
	}

	address := ETHAddress{}
	err = address.Set(c.Address)
	if err != nil {
		return errors.Wrapf(err, "parsing address argument")
	}
	amount := TRBAmount{}
	err = amount.Set(c.Amount)
	if err != nil {
		return errors.Wrapf(err, "parsing amount argument")
	}
	account, err := getAccountFor(accounts, c.Account)
	if err != nil {
		return err
	}
	return ops.Transfer(ctx, logger, client, contract, account, address.addr, amount.Int)

}

type approveCmd tokenCmd

func (c *approveCmd) Run() error {
	cfg, err := parseConfig(string(c.Config))
	if err != nil {
		return errors.Wrapf(err, "creating config")
	}

	logger := logging.NewLogger()

	ctx := context.Background()
	client, contract, accounts, err := createTellorVariables(ctx, logger, cfg.Ethereum)
	if err != nil {
		return errors.Wrapf(err, "creating tellor variables")
	}

	address := ETHAddress{}
	err = address.Set(c.Address)
	if err != nil {
		return errors.Wrapf(err, "parsing address argument")
	}
	amount := TRBAmount{}
	err = amount.Set(c.Amount)
	if err != nil {
		return errors.Wrapf(err, "parsing amount argument")
	}
	account, err := getAccountFor(accounts, c.Account)
	if err != nil {
		return err
	}
	return ops.Approve(ctx, logger, client, contract, account, address.addr, amount.Int)
}

type accountsCmd struct {
	Config configPath `type:"existingfile" help:"path to config file"`
}

func (a *accountsCmd) Run() error {
	cfg, err := parseConfig(string(a.Config))
	if err != nil {
		return errors.Wrapf(err, "creating config")
	}

	logger := logging.NewLogger()

	ctx := context.Background()
	_, _, accounts, err := createTellorVariables(ctx, logger, cfg.Ethereum)
	if err != nil {
		return errors.Wrapf(err, "creating tellor variables")
	}

	for i, account := range accounts {
		level.Info(logger).Log("msg", "account", "no", i, "address", account.Address.String())
	}

	return nil
}

type balanceCmd struct {
	Config  configPath `type:"existingfile" help:"path to config file"`
	Address string     `arg:"" optional:""`
}

func (b *balanceCmd) Run() error {
	cfg, err := parseConfig(string(b.Config))
	if err != nil {
		return errors.Wrapf(err, "creating config")
	}

	logger := logging.NewLogger()

	ctx := context.Background()
	client, contract, _, err := createTellorVariables(ctx, logger, cfg.Ethereum)
	if err != nil {
		return errors.Wrapf(err, "creating tellor variables")
	}

	addr := ETHAddress{}
	if b.Address == "" {
		err = addr.Set(contract.Address.String())
		if err != nil {
			return errors.Wrapf(err, "parsing argument")
		}
	} else {
		err = addr.Set(b.Address)
		if err != nil {
			return errors.Wrapf(err, "parsing argument")
		}
	}
	return ops.Balance(ctx, logger, client, contract, addr.addr)
}

type depositCmd struct {
	Config  configPath `type:"existingfile" help:"path to config file"`
	Account int        `arg:"" optional:""`
}

func (d depositCmd) Run() error {
	cfg, err := parseConfig(string(d.Config))
	if err != nil {
		return errors.Wrapf(err, "creating config")
	}

	logger := logging.NewLogger()

	ctx := context.Background()
	client, contract, accounts, err := createTellorVariables(ctx, logger, cfg.Ethereum)
	if err != nil {
		return errors.Wrapf(err, "creating tellor variables")
	}
	account, err := getAccountFor(accounts, d.Account)
	if err != nil {
		return err
	}
	return ops.Deposit(ctx, logger, client, contract, account)

}

type withdrawCmd struct {
	Config  configPath `type:"existingfile" help:"path to config file"`
	Address string     `arg:"" required:""`
	Account int        `arg:"" optional:""`
}

func (w withdrawCmd) Run() error {
	cfg, err := parseConfig(string(w.Config))
	if err != nil {
		return errors.Wrapf(err, "creating config")
	}

	logger := logging.NewLogger()

	ctx := context.Background()
	client, contract, accounts, err := createTellorVariables(ctx, logger, cfg.Ethereum)
	if err != nil {
		return errors.Wrapf(err, "creating tellor variables")
	}

	addr := ETHAddress{}
	err = addr.Set(w.Address)
	if err != nil {
		return errors.Wrapf(err, "parsing argument")
	}
	account, err := getAccountFor(accounts, w.Account)
	if err != nil {
		return err
	}
	return ops.WithdrawStake(ctx, logger, client, contract, account)

}

type requestCmd struct {
	Config  configPath `type:"existingfile" help:"path to config file"`
	Account int        `arg:"" optional:""`
}

func (r requestCmd) Run() error {
	cfg, err := parseConfig(string(r.Config))
	if err != nil {
		return errors.Wrapf(err, "creating config")
	}

	logger := logging.NewLogger()

	ctx := context.Background()
	client, contract, accounts, err := createTellorVariables(ctx, logger, cfg.Ethereum)
	if err != nil {
		return errors.Wrapf(err, "creating tellor variables")
	}
	account, err := getAccountFor(accounts, r.Account)
	if err != nil {
		return err
	}
	return ops.RequestStakingWithdraw(ctx, logger, client, contract, account)
}

type statusCmd struct {
	Config  configPath `type:"existingfile" help:"path to config file"`
	Account int        `arg:"" optional:""`
}

func (s statusCmd) Run() error {
	cfg, err := parseConfig(string(s.Config))
	if err != nil {
		return errors.Wrapf(err, "creating config")
	}

	logger := logging.NewLogger()

	ctx := context.Background()
	client, contract, accounts, err := createTellorVariables(ctx, logger, cfg.Ethereum)
	if err != nil {
		return errors.Wrapf(err, "creating tellor variables")
	}
	account, err := getAccountFor(accounts, s.Account)
	if err != nil {
		return err
	}
	return ops.ShowStatus(ctx, logger, client, contract, account)
}

type migrateCmd struct {
	Config configPath `type:"existingfile" help:"path to config file"`
}

func (s migrateCmd) Run() error {
	cfg, err := parseConfig(string(s.Config))
	if err != nil {
		return errors.Wrapf(err, "creating config")
	}

	logger := logging.NewLogger()

	ctx := context.Background()
	client, contract, accounts, err := createTellorVariables(ctx, logger, cfg.Ethereum)
	if err != nil {
		return errors.Wrapf(err, "creating tellor variables")
	}

	// Do migration for each account.
	for _, account := range accounts {
		level.Info(logger).Log("msg", "TRB migration", "account", account.Address.String())
		auth, err := ethereum.PrepareEthTransaction(ctx, client, account)
		if err != nil {
			return errors.Wrap(err, "prepare ethereum transaction")
		}

		tx, err := contract.Migrate(auth)
		if err != nil {
			return errors.Wrap(err, "contract failed")
		}
		level.Info(logger).Log("msg", "TRB migrated", "txHash", tx.Hash().Hex())
	}
	return nil
}

type newDisputeCmd struct {
	Config     configPath `type:"existingfile" help:"path to config file"`
	requestId  string     `arg:""  help:"the request id to dispute it"`
	timestamp  string     `arg:""  help:"the submitted timestamp to dispute"`
	minerIndex string     `arg:""  help:"the miner index to dispute"`
	Account    int        `arg:"" optional:""`
}

func (n newDisputeCmd) Run() error {
	// cfg, err := parseConfig(string(n.Config))
	// if err != nil {
	// 	return errors.Wrapf(err, "creating config")
	// }

	// logger := logging.NewLogger()

	// ctx := context.Background()
	// client, contract, accounts, err := createTellorVariables(ctx, logger, cfg.Ethereum)
	// if err != nil {
	// 	return errors.Wrapf(err, "creating tellor variables")
	// }

	// requestID := EthereumInt{}
	// err = requestID.Set(n.requestId)
	// if err != nil {
	// 	return errors.Wrapf(err, "parsing argument")
	// }
	// timestamp := EthereumInt{}
	// err = timestamp.Set(n.timestamp)
	// if err != nil {
	// 	return errors.Wrapf(err, "parsing argument")
	// }
	// minerIndex := EthereumInt{}
	// err = minerIndex.Set(n.minerIndex)
	// if err != nil {
	// 	return errors.Wrapf(err, "parsing argument")
	// }
	// account, err := getAccountFor(accounts, n.Account)
	// if err != nil {
	// 	return err
	// }
	// return ops.Dispute(ctx, logger, client, contract, account, requestID.Int, timestamp.Int, minerIndex.Int)

	return nil

}

type voteCmd struct {
	Config    configPath `type:"existingfile" help:"path to config file"`
	disputeId string     `arg:""  help:"the dispute id"`
	support   bool       `arg:""  help:"true or false"`
	Account   int        `arg:"" optional:""`
}

func (v voteCmd) Run() error {
	// cfg, err := parseConfig(string(v.Config))
	// if err != nil {
	// 	return errors.Wrapf(err, "creating config")
	// }

	// logger := logging.NewLogger()

	// ctx := context.Background()
	// client, contract, accounts, err := createTellorVariables(ctx, logger, cfg.Ethereum)
	// if err != nil {
	// 	return errors.Wrapf(err, "creating tellor variables")
	// }

	// disputeID := EthereumInt{}
	// err = disputeID.Set(v.disputeId)
	// if err != nil {
	// 	return errors.Wrapf(err, "parsing argument")
	// }
	// account, err := getAccountFor(accounts, v.Account)
	// if err != nil {
	// 	return err
	// }
	// return ops.Vote(ctx, logger, client, contract, account, disputeID.Int, v.support)
	return nil
}

type showCmd struct {
	Config  configPath `type:"existingfile" help:"path to config file"`
	Account int        `arg:"" optional:""`
}

func (s showCmd) Run() error {
	// cfg, err := parseConfig(string(s.Config))
	// if err != nil {
	// 	return errors.Wrapf(err, "creating config")
	// }

	// logger := logging.NewLogger()

	// ctx := context.Background()
	// client, contract, accounts, err := createTellorVariables(ctx, logger, cfg.Ethereum)
	// if err != nil {
	// 	return errors.Wrapf(err, "creating tellor variables")
	// }
	// account, err := getAccountFor(accounts, s.Account)
	// if err != nil {
	// 	return err
	// }
	// return ops.List(ctx, cfg, logger, client, contract, account)
	return nil
}

type dataserverCmd struct {
	Config configPath `type:"existingfile" help:"path to config file"`
}

func (d dataserverCmd) Run() error {
	// cfg, err := parseConfig(string(d.Config))
	// if err != nil {
	// 	return errors.Wrapf(err, "creating config")
	// }

	// When the ServerWhitelist is empty try to get
	//for _, acc := range accounts {
	// 	cfg.DataServer.ServerWhitelist = append(cfg.DataServer.ServerWhitelist, acc.Address.String())
	// }

	// logger := logging.NewLogger()

	// ctx := context.Background()
	// client, contract, accounts, err := createTellorVariables(ctx, logger, cfg.Ethereum)
	// if err != nil {
	// 	return errors.Wrapf(err, "creating tellor variables")
	// }

	// DB, err := migrateAndOpenDB(logger, cfg)
	// if err != nil {
	// 	return errors.Wrapf(err, "initializing database")
	// }
	// proxy, err := db.OpenLocal(logger, cfg, DB)
	// if err != nil {
	// 	return errors.Wrapf(err, "open remote DB instance")
	// }
	// ds, err := dataServer.NewDataServerOps(ctx, logger, cfg, proxy, client, contract, accounts)
	// if err != nil {
	// 	return errors.Wrapf(err, "creating data server")
	// }
	// // Start and wait for it to be ready.
	// if err := ds.Start(); err != nil {
	// 	return errors.Wrap(err, "starting data server")
	// }

	// // We define our run groups here.
	// var g run.Group
	// // Run groups.
	// {
	// 	// Handle interupts.
	// 	g.Add(run.SignalHandler(context.Background(), syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM))

	// 	// Metrics server.
	// 	{
	// 		http.Handle("/metrics", promhttp.Handler())
	// 		srv, err := rest.Create(logger, cfg, ctx, proxy, cfg.Mine.ListenHost, cfg.Mine.ListenPort)
	// 		if err != nil {
	// 			return errors.Wrapf(err, "creating http data server")
	// 		}
	// 		g.Add(func() error {
	// 			level.Info(logger).Log("msg", "starting metrics server", "addr", cfg.Mine.ListenHost, "port", cfg.Mine.ListenPort)
	// 			// returns ErrServerClosed on graceful close
	// 			var err error
	// 			if err = srv.Start(); err != nil {
	// 				err = errors.Wrapf(err, "ListenAndServe")
	// 			}
	// 			level.Info(logger).Log("msg", "metrics server shutdown complete")
	// 			return err
	// 		}, func(error) {
	// 			if srv.Stop() != nil {
	// 				level.Error(logger).Log("msg", "shutting down the rest service", "err", err)
	// 			}
	// 		})
	// 	}

	// }

	// if err := g.Run(); err != nil {
	// 	level.Info(logger).Log("msg", "main exited with error", "err", err)
	// 	return err
	// }

	// level.Info(logger).Log("msg", "main shutdown complete")
	return nil
}

type mineCmd struct {
	Config configPath `type:"existingfile" help:"path to config file"`
}

func (m mineCmd) Run() error {
	// Defining a global context for starting and stopping of components.
	ctx := context.Background()
	cfg, err := parseConfig(string(m.Config))
	if err != nil {
		return errors.Wrapf(err, "creating config")
	}

	logger := logging.NewLogger()
	client, contract, accounts, err := createTellorVariables(ctx, logger, cfg.Ethereum)
	if err != nil {
		return errors.Wrapf(err, "creating tellor variables")
	}

	// DataServer is the Telliot data server.
	// var proxy db.DataServerProxy

	proxy, err := db.Open(logger, cfg.Db)
	if err != nil {
		return errors.Wrapf(err, "opening DB instance")
	}
	// if cfg.Mine.RemoteDBHost != "" {
	// 	proxy, err = db.OpenRemote(logger, cfg, DB)
	// } else {
	// proxy, err = db.OpenLocal(logger, cfg, DB)
	// }
	// if err != nil {
	// 	return errors.Wrapf(err, "open remote DB instance")

	// }

	// We define our run groups here.
	var g run.Group
	// Run groups.
	{
		// Handle interupts.
		g.Add(run.SignalHandler(context.Background(), syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM))

		// Open the TSDB database.
		// TODO when remote use NewSampleAndChunkQueryableClient.
		tsdbOptions := tsdb.DefaultOptions()
		// 2 days are enough as the agregator needs data only 24 hours in the past.
		tsdbOptions.RetentionDuration = int64(2 * 24 * time.Hour)
		tsDB, err := tsdb.Open(cfg.Db.Path, nil, nil, tsdbOptions)
		if err != nil {
			return errors.Wrapf(err, "creating tsdb DB")
		}

		defer func() {
			if err := tsDB.Close(); err != nil {
				level.Error(logger).Log("msg", "closing the tsdb", "err", err)
			}
		}()

		// Web/Api server.
		{
			srv := web.New(logger, ctx, tsDB, cfg.Web)
			g.Add(func() error {
				err := srv.Start()
				level.Info(logger).Log("msg", "web server shutdown complete")
				return err
			}, func(error) {
				srv.Stop()
			})
		}

		// Aggregator.
		if cfg.Aggregator.ConfidIntvThreshold.Duration == 0 {
			// Values outside the default tracker interval are not used and decrease the confidence level.
			cfg.Aggregator.ConfidIntvThreshold.Duration = cfg.IndexTracker.Interval.Duration + time.Second
		}
		aggregator, err := aggregator.New(logger, ctx, cfg.Aggregator, tsDB, client)
		if err != nil {
			return errors.Wrapf(err, "creating aggregator")
		}

		g.Add(func() error {
			err := aggregator.Run()
			level.Info(logger).Log("msg", "aggregator shutdown complete")
			return err
		}, func(error) {
			aggregator.Stop()
		})

		// Index tracker.
		// TODO Run only when not using remote DB
		index, err := index.New(logger, ctx, cfg.IndexTracker, tsDB, client)
		if err != nil {
			return errors.Wrapf(err, "creating index tracker")
		}

		g.Add(func() error {
			err := index.Run()
			level.Info(logger).Log("msg", "index shutdown complete")
			return err
		}, func(error) {
			index.Stop()
		})

		// Profit tracker.
		var accountAddrs []common.Address
		for _, acc := range accounts {
			accountAddrs = append(accountAddrs, acc.Address)
		}
		profitTracker, err := profit.NewProfitTracker(logger, ctx, cfg.ProfitTracker, client, contract, proxy, accountAddrs)
		if err != nil {
			return errors.Wrapf(err, "creating profit tracker")
		}
		g.Add(func() error {
			err := profitTracker.Start()
			level.Info(logger).Log("msg", "tasker shutdown complete")
			return err
		}, func(error) {
			profitTracker.Stop()
		})

		// Event tasker.
		// tasker, taskerChs, err := tasker.NewTasker(ctx, logger, cfg.Tasker, client, contract, accounts)
		// if err != nil {
		// 	return errors.Wrapf(err, "creating tasker")
		// }
		// g.Add(func() error {
		// 	err := tasker.Start()
		// 	level.Info(logger).Log("msg", "tasker shutdown complete")
		// 	return err
		// }, func(error) {
		// 	tasker.Stop()
		// })

		// Create a submitter for each account.
		// gasPriceTracker := gasPrice.New(logger, client)
		// for _, account := range accounts {
		// 	transactor, err := transactor.NewTransactor(logger, cfg, gasPriceTracker, client, account, contract)
		// 	if err != nil {
		// 		return errors.Wrapf(err, "creating transactor")
		// 	}
		// 	// Get a channel on which it listens for new data to submit.
		// 	submitter, submitterCh, err := submitter.NewSubmitter(ctx, cfg, logger, client, contract, account, proxy, transactor, gasPriceTracker)
		// 	if err != nil {
		// 		return errors.Wrapf(err, "creating submitter")
		// 	}
		// 	g.Add(func() error {
		// 		err := submitter.Start()
		// 		level.Info(logger).Log("msg", "submitter shutdown complete",
		// 			"addr", account.Address.String(),
		// 		)
		// 		return err
		// 	}, func(error) {
		// 		submitter.Stop()
		// 	})

		// 	// Will be used to cancel pending submissions.
		// 	tasker.AddSubmitCanceler(submitter)

		// 	// the Miner component.
		// 	miner, err := mining.NewMiningManager(logger, ctx, cfg, proxy, contract, taskerChs[account.Address.String()], submitterCh, account, client)
		// 	if err != nil {
		// 		return errors.Wrapf(err, "creating miner")
		// 	}
		// 	g.Add(func() error {
		// 		err := miner.Start()
		// 		level.Info(logger).Log("msg", "miner shutdown complete",
		// 			"addr", account.Address.String(),
		// 		)
		// 		return err
		// 	}, func(error) {
		// 		miner.Stop()
		// 	})
		// }
	}

	if err := g.Run(); err != nil {
		level.Error(logger).Log("msg", "main exited with error", "err", err)
		return err
	}

	level.Info(logger).Log("msg", "main shutdown complete")
	return nil
}

func getAccountFor(accounts []*ethereum.Account, accountNo int) (*ethereum.Account, error) {
	if accountNo < 0 || accountNo >= len(accounts) {
		return nil, errors.New("account not found")
	}
	return accounts[accountNo], nil
}
