package core 

import (
  "context"
)

type msgAtomic struct {
  ID string
  err error
}

func finaliseAtomic(ctx context.Context, ch chan<-msgAtomic, id string, err *error) {
  newMsg := msgAtomic {
    ID: id,
    err: (*err),
  }
  ctx.Done()
  ch<-newMsg
}

func max(a, b int) int {
  if a > b { return a }
  return b
}

func min(a, b int) int {
  if a < b { return a }
  return b
}
