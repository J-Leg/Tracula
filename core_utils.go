package tracula 

import (
  "context"
)

func finaliseAtomic(ctx context.Context, ch chan<-msgAtomic, id string, err *error) {
  newMsg := msgAtomic {
    ID: id,
    err: (*err),
    ctx: ctx,
  }
  ctx.Done()
  ch<-newMsg
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
