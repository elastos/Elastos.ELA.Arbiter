Elastos.ELA.Arbiter version 0.2.0 is now available from:

  <https://download.elastos.org/elastos-arbiter/elastos-arbitier-v0.2.0/>

This is a new minor version release.

Please report bugs using the issue tracker at GitHub:

  <https://github.com/elastos/Elastos.ELA.Arbiter/issues>

How to Upgrade
==============

If you are running version release_v0.1.2 and before, you should shut it down and wait until
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

Support CR council member claim DPOS node.

0.2.0 change log
=================

Detailed release notes follow
 
- #14 Modify mainnet spv seed list
- #13 Set default value for config
- #12 Add prefix of node version
- #11 Add NewP2PProtocolVersionHeight
- #10 Merge remote-tracking branch 'elastos/release_v0.1.2' into dev
- #9 add getarbiterpeersinfo rpc and set default value for config
- #8 Add new connected peers to addr
- #7 Fix nil exception
- #6 Fix arbiter onduty in a row when claim dpos node started
- #5 Update version of SPV
- #4 Modify to support inactived or impeached arbitrator
- #3 Fixed an issue that withdraw from side chain transaction check failed
- #2 Modify to get peers correctly
- #1 Make crypto.SignatureScriptLength error as warning
