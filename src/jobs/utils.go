package jobs

import (
	"com/logs"
	"context"
)

func CreateJob(ctx context.Context, jobFunction func(ctx context.Context) error, jobDescription string) func() {
	return func() {
		logger, err := logs.Logger(ctx).CreateChildJob(ctx, logs.Import)
		if err != nil {
			logs.ErrorWithContext(ctx, "unable to create child job: %v", err)
			return
		}
		jobctx := logs.ContextWithLogger(ctx, logger)
		logger.Info(jobctx, "storing all data from YoLink connection...")
		
		err = jobFunction(jobctx)
		if err != nil {
			logger.Error(jobctx, "error while storing sensor data: %v", err)
			logger.End(jobctx)
			return
		}
		logger.Info(jobctx, "job ending normally")
		logger.End(jobctx)
	}
}