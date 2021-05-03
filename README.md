# Single executor

A hobby project written in Go.

**Please note that this is a work in progress, so the code is not well tested/documented, and there are many improvements that could be made to maintainability, performance and security.**

## The problem

How do we keep a process running on a single machine in a highly-available solution.
The following assumptions are made:
* It is _critical_ that the process only ever runs on _at most one_ machine at a time.
* The process being run can be any arbitrary binary, but we must not modify it.
* It is okay for the process to not be running during fail-over, but this downtime should be mitigated.
* The system should automatically self-correct.
* The nodes are all "trusted"; non-BFT is not required.

## The solution

Our solution to this problem is an adaption of 
[Raft leader election](https://en.wikipedia.org/wiki/Raft_(algorithm)#Leader_election).

The configured nodes will elect a leader amongst themselves. When a leader
is elected, the process will start on that node. Note that we require a majority
across the whole network for a leader to be elected, so in the case of a 50/50 split
brain problem, no leader will be elected.

Leadership is temporary, for as long as the leader receives heartbeats from the majority
of its followers (again, a cluster-wide majority must be reached), it will refresh its leadership
timeout. If this majority is not reached within a given time window, leadership will be 
relinquished, and a new election will start.

Whilst leadership is held, the accompanying binary will be run on that node.
Some grace periods will be granted between starting/stopping the process
so that we never overlap execution. Note this is not yet implemented.

### Known Limitations

* Static configuration. The network needs to be brought down to add/remove nodes.
* The system handles up to 50% node failures. If more than 50% of the connected
  nodes fail, the binary will not run.
* Non-BFT. This solution assumes there can be no bad actors.
* Transport is currently insecure.

### Future additions

* A separate node monitoring component, that can force a watchdog instance
  to drop leader when the network/node is not healthy.
* Support for two modes of recovery:
  * Automatic: as with the current solution, the nodes will automatically elect a new leader when one goes down.
    In this configuration, the system is self-healing.
  * Manual: the watchdog instances will require manual activation
    to leadership state. The network will still hold election, but in this
    configuration, it would require an active confirmation from a human
    to proceed to an active leader state. In this configuration, there is more
    control of the system. This would be useful to test/prove the automatic recovery
    system before switching to it.
* Revise the network transport protocols. Currently, we have two main mechanisms:
  * `HTTP` - these channels are simply for demonstration/dashboard purposes, such as JSON responses to watchdog state,
    or commands to blacklist a network or kill a watchdog instance.
  * `raw UDP` - the underlying protocol used by the watchdog instances to communicate with one another. In the future, this would
    probably want to be replaced with some kind of RPC for reliability, security & stability.
  

## Components

`watchdog` is the core component that implements the distributed algorithm. Note that
there are some concepts in this component to allow demonstration (such as app-level blacklisting
of other nodes in the network to simulate network connectivty issues/split-brain problem).

The other components in this repo are present to facilitate development and demonstration
of the core `watchdog` component. These are:
* `dashboard` - a simple HTTP application that displays info of the system state
* `chain` - a node which emulates an external system/node (such as a blockchain).
  This simply listen for incoming signatures and records them. The signing history
  can be seen in JSON via an HTTP call.
  Note this is not hooked up in the test system/dashboard yet.
* `binary` - a simple producer for the `chain`. For demonstration purposes, this
  is the executable that we only ever want to be running once. In a real system, this
  could be any executable. For now, we can observe the signing history on `chain` to test
  whether or not one binary is running.
  Note this is not hooked up in the test system/dashboard yet.
  
We use docker throughout this to simulate nodes in a network. Using docker compose,
we easily have each component running in its own container(s) to simulate a multi-node environment on a local machine.
In reality, these components would likely be deployed across data centers, but here docker gives us
an realistic test environment.

## Installation

Requirements:
* `GNU make` (developed on v4.2.1)
* `docker` (developed on v20.10.5)
* `docker-compose` (developed on 1.29.0)

To compile everything and bring up a test system with dashboard,
```
make run-demo
```

Then navigate to `http://localhost:8081/dashboard`.

### Dashboard

Each watchdog instance state is displayed,

![Dashboard preview 0](doc/dashboard-preview-0.png)

The Network page contains a topological display of the network, and allows for starting/stopping
individual nodes in the network.

![Dashboard preview 1](doc/dashboard-preview-1.png)

Here we have hit `Stop` on the leader (`5`), and the network has resolved a new leader (`3`),

![Dashboard preview 2](doc/dashboard-preview-2.png)

We can also enable/disable individual network links between nodes. In this case, we have broken
links between the leader (`3`) and some of its followers. Because `3` can no longer reached a majority
consensus on its leadership, it relinquishes leader state and `4` wins the next election,

![Dashboard preview 3](doc/dashboard-preview-3.png)

## Development

Some useful commands for development.

To bring up a container for developing the VueJS dashboard,

```
make dashboardbuilder
```

Then once inside this terminal, you have access to `npm` for managing, developing and
building the app.

```
# to compile the app
npm run build

# to compile and watch for changes
npm run watch
```
