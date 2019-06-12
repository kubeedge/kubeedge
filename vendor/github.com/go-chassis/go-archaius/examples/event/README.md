event.go keep a file under archaius's management, and watch file changes,
so that if there is change in file, 
the event will be triggered, and listener will receive the event

```
go build event.go
./event
```

change **age** config in event.yaml 

check the stdout to see events