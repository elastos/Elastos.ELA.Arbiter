Elastos.ELA.Arbiter version 0.2.2 is now available from:

  <https://download.elastos.org/elastos-arbiter/elastos-arbitier-v0.3.0/>

This is a new minor version release.

Please report bugs using the issue tracker at GitHub:

  <https://github.com/elastos/Elastos.ELA.Arbiter/issues>

How to Upgrade
==============

If you are running version release_v0.2.1 and before, you should shut it down and wait until
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

Ignore WithdrawFromSideChain transaction with no CrossChainAssets.

0.2.2 change log
=================

Detailed release notes follow
 
- #24 Modify to create cross chain transaction correctly
- #25 Set default value of MaxRedeemScriptDataSize
- #31 Set default value of DPOSNodeCrossChainHeight
