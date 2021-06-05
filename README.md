# Subtrate Crond

Daemon to execute scheduled jobs based on Substrate activity

## About 

[Substrate](https://substrate.dev/) is a modular framework that enables the creation of new blokchains by composing custom pre-build components, and it is the foundation on which some of the most important recent blockchains, such as [Polkadot](https://polkadot.network/) and [Kusama](https://kusama.network/), are built.

This tool monitors the activity on Substrate based blockchains and enables the scheduling of jobs based on that activity in a very similar way to what you do with conventional cron. You can schedule tasks to be executed at the beginning of each era or session, or even each block. This can be useful, for example, for validator operators to automate rewards payouts by triggering tools such as [substrate-payctl](https://github.com/stakelink/substrate-payctl).


## Install

Install it using _go get_;

```
go get github.com/stakelink/substrate-crond
```

## Usage

After installig it _substrate-crond_ executable should be available on the system.

```
$ substrate-crond
NAME:
   substrate-crond - Daemon to execute scheduled jobs based on Substrate activity

USAGE:
   substrate-crond  [global options] command [command options] [arguments...]

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --rpc-url value, -r value       (default: "wss://rpc.polkadot.io")
   --crontab-file value, -c value  (default: "/etc/substrate-crond/crontab")
   --daemon, -d                    (default: false)
   --help, -h                      show help (default: false)
```

### Configuration 

The configuration is based on a crontab file contains instructions to the cron daemon of the general form: ''run this command every slot''. The format of an entry is pretty much the same than the a regular crontab file, but instead of minutes, hours and days, we specify slots, sessions and eras. And in the same way, several specifications are allowed:

	(*) A field may be an asterisk, which stands as ANY.

	(-) A field may contain a script symbol to indicate a range.

	(,) A field may contain a coma to aggregate expressions.

	(/) A field may be complemented with an slash symbol to trigger only under a modulus.

The supported fields at the moment are;

 * Local Slot: It is the local slot within a session (it means the first slot of a session is 0).
 * Local Session: It is the local session within a era (it means the first session of an era is 0).
 * Era: It is the era.


### Examples

Here are some possible uses by way of example:

```
*       *    *     echo "this is executed on evey slot (block)"
*/10    *    *     echo "this is executed every tens slots (blocks)"
0       *    *     echo "this is executed on the first block of each session (once per session)"
0       0    *     echo "this is executed on the first block of each era (once per era)"
0       0    */2   echo "this is executed on the first block every two eras"
0-9,15  *    *     echo "this is executed on the first 10 bocks and the 15th block, for all sessions"
```

