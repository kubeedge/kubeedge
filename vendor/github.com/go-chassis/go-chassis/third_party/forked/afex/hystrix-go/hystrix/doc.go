/*
Package hystrix is a latency and fault tolerance library designed to isolate
points of access to remote systems, services and 3rd party libraries, stop
cascading failure and enable resilience in complex distributed systems where
failure is inevitable.

Based on the java project of the same name, by Netflix. https://github.com/Netflix/Hystrix

Execute code as a Hystrix command

Define your application logic which relies on external systems, passing your function to Go. When that system is healthy this will be the only thing which executes.

	hystrix.Go("my_command", func() error {
		// talk to other services
		return nil
	}, nil)

Defining fallback behavior

If you want code to execute during a service outage, pass in a second function to Go. Ideally, the logic here will allow your application to gracefully handle external services being unavailable.

This triggers when your code returns an error, or whenever it is unable to complete based on a variety of health checks https://github.com/Netflix/Hystrix/wiki/How-it-Works.

	hystrix.Go("my_command", func() error {
		// talk to other services
		return nil
	}, func(err error) error {
		// do this when services are down
		return nil
	})

Waiting for output

Calling Go is like launching a goroutine, except you receive a channel of errors you can choose to monitor.

	output := make(chan bool, 1)
	errors := hystrix.Go("my_command", func() error {
		// talk to other services
		output <- true
		return nil
	}, nil)

	select {
	case out := <-output:
		// success
	case err := <-errors:
		// failure
	}

Synchronous API

Since calling a command and immediately waiting for it to finish is a common pattern, a synchronous API is available with the Do function which returns a single error.

	err := hystrix.Do("my_command", func() error {
		// talk to other services
		return nil
	}, nil)

Configure settings

During application boot, you can call ConfigureCommand to tweak the settings for each command.

	hystrix.ConfigureCommand("my_command", hystrix.CommandConfig{
		Timeout:               1000,
		MaxConcurrentRequests: 100,
		ErrorPercentThreshold: 25,
	})

You can also use Configure which accepts a map[string]CommandConfig.

Enable dashboard metrics

In your main.go, register the event stream HTTP handler on a port and launch it in a goroutine.  Once you configure turbine for your Hystrix Dashboard https://github.com/Netflix/Hystrix/tree/master/hystrix-dashboard to start streaming events, your commands will automatically begin appearing.

	hystrixStreamHandler := hystrix.NewStreamHandler()
	hystrixStreamHandler.Start()
	go http.ListenAndServe(net.JoinHostPort("", "81"), hystrixStreamHandler)
*/
package hystrix
