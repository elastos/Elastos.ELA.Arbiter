Elastos.ELA.Arbiter version 0.2.3 is now available from:

  <https://download.elastos.org/elastos-arbiter/elastos-arbitier-v0.2.3/>

This is a new minor version release.

Please report bugs using the issue tracker at GitHub:

  <https://github.com/elastos/Elastos.ELA.Arbiter/issues>

How to Upgrade
==============

If you are running version release_v0.2.2 and before, you should shut it down and wait until
 it has completely closed, then just copy over `arbiter` (on Linux).

However, as usual, config, keystore and chaindata files are compatible.

Compatibility
==============

Elastos.ELA.Arbiter is supported and extensively tested on operating systems
using the Linux kernel. It is not recommended to use Elastos.ELA.Arbiter on
unsupported systems.

Elastos.ELA.Arbiter should also work on most other Unix-like systems but is not
as frequently tested on them.

As previously-supported CPU platforms, this release's pre-compiled
distribution provides binaries for the x86_64 platform.

Notable changes
===============

Support to start with wallet path
Support to use -v to print version information.

0.2.3 change log
=================

Detailed release notes follow
 
- #41 Change default value of SyncStartHeight in sample file
- #41 Support -v to print version information
- #40 Only Pow chain need to divide and support -v to print version information
- #39 Make wallet path configurable
