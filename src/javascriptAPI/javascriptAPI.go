package javascriptAPI

import (
	"time"

	"rogchap.com/v8go"

	"src/config"
)

func Javascript_run_isolate(vm *v8go.Isolate, script string, mode string, args ...[2]string) (string, error) {
	vals := make(chan string, 1)
	errs := make(chan error, 1)
	delay_fns := []func(){}

	tmp_store := map[string]interface{}{}

	go func() {

		ctx := Javascript_context_init(vm, errs, &delay_fns, tmp_store, mode, args...) // initiallize context api and functions
		defer ctx.Close()

		val, err := ctx.RunScript(script+";returning_print_buffer", "main.js") // exec a long running script
		if err != nil {
			errs <- err
			return
		}
		vals <- val.String()
	}()

	select {
	case val := <-vals:
		// sucess

		return val, nil
	case err := <-errs:
		// javascript error
		for _, fn := range delay_fns {
			fn()
		}
		return "", err
	case <-time.After(time.Duration(config.Javascript_timeout) * time.Millisecond): // get the Isolate from the context
		for _, fn := range delay_fns {
			fn()
		}
		err := <-errs // will get a termination error back from the running script
		return "", err
	}
}
