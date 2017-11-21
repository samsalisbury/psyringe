# psyringe dev notes

Try using golang.org/x/sync/singleflight to guarantee constructors are only called once.
This will need a bit of additional logic as singleflight only guarantees that a func
is not being run concurrently but allows it to be run more than once in total.
