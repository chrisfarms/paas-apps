# go-selfdos

## Usage:

Don't use it. Seriously.

## But really:

```
$ cf push bad-actor

...wait for deploy...

$ cf scale -i 20

...wait for it to warm up...

$ cf appa

FAILED

$ cf login

FAILED

$ cf delete -f -r bad-actor
$ cf delete -f -r bad-actor
$ cf delete -f -r bad-actor
$ cf delete -f -r bad-actor
$ cf delete -f -r bad-actor
$ cf delete -f -r bad-actor
$ cf delete -f -r bad-actor
OK

... everything fixed...
```

