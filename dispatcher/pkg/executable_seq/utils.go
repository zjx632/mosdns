//     Copyright (C) 2020-2021, IrineSistiana
//
//     This file is part of mosdns.
//
//     mosdns is free software: you can redistribute it and/or modify
//     it under the terms of the GNU General Public License as published by
//     the Free Software Foundation, either version 3 of the License, or
//     (at your option) or later version.
//
//     mosdns is distributed in the hope that it will be useful,
//     but WITHOUT ANY WARRANTY; without even the implied warranty of
//     MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//     GNU General Public License for more details.
//
//     You should have received a copy of the GNU General Public License
//     along with this program.  If not, see <https://www.gnu.org/licenses/>.

package executable_seq

import (
	"context"
	"errors"
	"github.com/IrineSistiana/mosdns/dispatcher/handler"
	"go.uber.org/zap"
)

// ExecRoot executes the ExecutableCmd and qCtx.ExecDefer().
// This intends for root cmd node.
func ExecRoot(ctx context.Context, qCtx *handler.Context, logger *zap.Logger, entry ExecutableCmd) error {
	_, err := entry.ExecCmd(ctx, qCtx, logger)
	if err == nil {
		err = qCtx.ExecDefer(ctx)
	}
	return err
}

func asyncWait(ctx context.Context, qCtx *handler.Context, logger *zap.Logger, c chan *parallelECSResult, total int) error {
	for i := 0; i < total; i++ {
		select {
		case res := <-c:
			if res.err != nil {
				logger.Warn("sequence failed", qCtx.InfoField(), zap.Int("sequence", res.from), zap.Error(res.err))
				continue
			}

			if res.r == nil {
				logger.Debug("sequence returned with an empty response", qCtx.InfoField(), zap.Int("sequence", res.from))
				continue
			}

			logger.Debug("sequence returned a response", qCtx.InfoField(), zap.Int("sequence", res.from))
			qCtx.SetResponse(res.r, res.status)
			return nil

		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// No response
	qCtx.SetResponse(nil, handler.ContextStatusServerFailed)
	return errors.New("no response")
}
