Elastos.ELA.Arbiter version 0.3.0 is now available from:

  <https://download.elastos.org/elastos-arbiter/elastos-arbitier-v0.3.0/>

This is a new minor version release.

Please report bugs using the issue tracker at GitHub:

  <https://github.com/elastos/Elastos.ELA.Arbiter/issues>

How to Upgrade
==============

If you are running version release_v0.2.3 and before, you should shut it down and wait until
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

Supports cross-chain data transfer.
Support automated cross-chain failure handling.
Support small cross-chain transaction transfers quickly.
Support register side-chain proposals.

0.3.0 change log
=================

Detailed release notes follow
 
- #62 Fixed an issue of register side chain
- #60 Fixed an issue of config file
- #59 Fix genesishash reverse problem
- #57 RegisteredSidechain takes effect after effective height
- #55 Get exchangeRate from RegisterSidechain proposal
- #54 Store registered sidechain info to config.json
- #53 Change ReturnDepositTxs from FinishedDB to SideChainDB
- #51 Add MaxNodePerHost config
- #45 Fixed some issue of new withdraw transaction 
- #44 Support register sidechian
- #43 Add invalid withdraw transaction monitor


