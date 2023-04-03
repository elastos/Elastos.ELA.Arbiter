Elastos.ELA.Arbiter version 0.4.0 is now available from:

  <https://download.elastos.org/elastos-arbiter/elastos-arbitier-v0.4.0/>

This is a new minor version release.

Please report bugs using the issue tracker at GitHub:

  <https://github.com/elastos/Elastos.ELA.Arbiter/issues>

How to Upgrade
==============

If you are running version release_v0.3.3 and before, you should shut it down and wait until
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

Support NFT generation.

0.4.0 change log
=================

Detailed release notes follow

### Bug Fixes

* **dpos2.0:** add schnorr type withdraw payload version check into consideration ([9611d8f](https://github.com/elastos/Elastos.ELA.Arbiter/commit/9611d8ffc7628b5fed4d439dcfab0f08a4e03938))
* **dpos2.0:** fixed ci error ([0b023db](https://github.com/elastos/Elastos.ELA.Arbiter/commit/0b023db7695f8d7b9bb3f4a93762644a3b9417da))
* **dpos2.0:** get GenesisBlockAddress based tx payload version ([e1fc256](https://github.com/elastos/Elastos.ELA.Arbiter/commit/e1fc2564a3c5c16f8d704a1a5a3113bf9796c649))
* **dpos2.0:** get GenesisBlockAddress correctly ([918c7ce](https://github.com/elastos/Elastos.ELA.Arbiter/commit/918c7cedef1957854651b5a9b7a68a9bef62a144))
* **dpos2.0:** get systemid correctly ([ca17868](https://github.com/elastos/Elastos.ELA.Arbiter/commit/ca178686f3e0b818451c72da7b4d35a49356f24a))
* **dpos2.0:** modify to calculate confirm count correctly ([3faffcc](https://github.com/elastos/Elastos.ELA.Arbiter/commit/3faffcca30f0ae7e4709b805bb26bb0eeb03b14e))
* **dpos2.0:** tiny fix ([69b7fba](https://github.com/elastos/Elastos.ELA.Arbiter/commit/69b7fba0d57e1fc8bf5f59fedc8262dc4ca7ebd4))

### Features

* **dpos2.0:** add default value of regnet ([022fe67](https://github.com/elastos/Elastos.ELA.Arbiter/commit/022fe670c67170b5252c57b1272ac75af80b34fc))
* **dpos2.0:** add default value of testnet config ([3bca902](https://github.com/elastos/Elastos.ELA.Arbiter/commit/3bca902e455452d02609da286a56b61cec0ec7a0))
* **dpos2.0:** add node version to rpc ([cb31b0d](https://github.com/elastos/Elastos.ELA.Arbiter/commit/cb31b0d843dd095dba25841688f6cebfd325fdb4))
* **dpos2.0:** change PingNonce from randomUint64 to bestHeight ([279a925](https://github.com/elastos/Elastos.ELA.Arbiter/commit/279a92560b86a2d0df4c36300da2cfae91021b63))
* **dpos2.0:** check if sender in current arbiters list ([6018c74](https://github.com/elastos/Elastos.ELA.Arbiter/commit/6018c74820182588c0e240e9a16a972ed3f697c1))
* **dpos2.0:** fixed an issue that reorganize may lead to panic ([9facb57](https://github.com/elastos/Elastos.ELA.Arbiter/commit/9facb57fd57e39495ab949170e1335fe0ac3994c))
* **dpos2.0:** ignore empty string arbiter ([2349f7d](https://github.com/elastos/Elastos.ELA.Arbiter/commit/2349f7d77e0ab69682001a676cd9dbe763452958))
* **dpos2.0:** initialize ELA repo functions ([0e8bcb0](https://github.com/elastos/Elastos.ELA.Arbiter/commit/0e8bcb0b07dcdb436933a8b51637f63c514f4ce2))
* **dpos2.0:** make DPoSV2StartHeight configurable ([705f897](https://github.com/elastos/Elastos.ELA.Arbiter/commit/705f8973e5e765b77a329d847e76eae212f68564))
* **dpos2.0:** modify to register side chain correctly ([11f0187](https://github.com/elastos/Elastos.ELA.Arbiter/commit/11f0187f69b1f9863dce2dc15b1845eb60c41845))
* **dpos2.0:** modify to save config.json correctly ([73bc77c](https://github.com/elastos/Elastos.ELA.Arbiter/commit/73bc77cc60848ddc8df1bcef352ded454f8b1b5a))
* **dpos2.0:** set PowChain to false in mainnet_config.json.sample ([bbca29c](https://github.com/elastos/Elastos.ELA.Arbiter/commit/bbca29c011c27aa8c6ad0381a00101137e719ca0))
* **dpos2.0:** set side chain name by registerSideChain transaction payload ([40186d7](https://github.com/elastos/Elastos.ELA.Arbiter/commit/40186d7474608ef3b460ac96e780caa72814c351))
* **dpos2.0:** support new stake related transactions ([7b30b45](https://github.com/elastos/Elastos.ELA.Arbiter/commit/7b30b45685af46df18dd740d578da5ac35b8ca03))
* **dpos2.0:** update config sample ([69498c2](https://github.com/elastos/Elastos.ELA.Arbiter/commit/69498c26619a415104953861e2ff68fc36d1314c))
* **dpos2.0:** update Elastos.ELA from v0.8.2 to dpos2.0 ([b3ef7e0](https://github.com/elastos/Elastos.ELA.Arbiter/commit/b3ef7e077a29d8211217f25717dcd348dd04d62d))
* **dpos2.0:** update go.mod ([7bf4688](https://github.com/elastos/Elastos.ELA.Arbiter/commit/7bf4688c219c0dc7fd5f5f786fec048b95d6154f))
* **dpos2.0:** update go.mod ([afc0006](https://github.com/elastos/Elastos.ELA.Arbiter/commit/afc00066e907d6633bdcb6757fbffffc3ff4f6c6))
* **dpos2.0:** update mainnet_config.json.sample ([62e7954](https://github.com/elastos/Elastos.ELA.Arbiter/commit/62e7954eb7073d6d967e7f221714939e09e313fd))
* **dpos2.0:** update schnorr code recording to btc witness script ([afd165e](https://github.com/elastos/Elastos.ELA.Arbiter/commit/afd165e6cd45766a6dfa6fdeda7204594b6da9c3))
* **dpos2.0:** update SideHeightInfo table ([a92a546](https://github.com/elastos/Elastos.ELA.Arbiter/commit/a92a5465ef8001adc5ad90f94b95667166caf750))
* **nft:** add Destroy NFT transaction ([0948f5e](https://github.com/elastos/Elastos.ELA.Arbiter/commit/0948f5eee8f0484e0aeca9defd5fa4b4715e96df))
* **nft:** add rpc  getPledgeBillBurnTransactionByHeight ([e4d033f](https://github.com/elastos/Elastos.ELA.Arbiter/commit/e4d033fb1920702455afd3eb3b478c330f126466))
* **nft:** change from GenesisBlockAddress to GenesisBlock ([1fc3278](https://github.com/elastos/Elastos.ELA.Arbiter/commit/1fc32781740b41acbca0746b7ffba3988bc4c071))
* **nft:** change NFTDestroyFromSideChain GenesisBlockHash ([a2b5a1e](https://github.com/elastos/Elastos.ELA.Arbiter/commit/a2b5a1e6a6abba8d2622ea90a78bc5ff7e4a1f83))
* **nft:** change RemoveSideChainTxs db into NFTDestroyTxs ([3ebe163](https://github.com/elastos/Elastos.ELA.Arbiter/commit/3ebe16361ea3a8aca0ede1f38e5802f9ae68aca1))
* **nft:** test nft destroy finished ([1144e14](https://github.com/elastos/Elastos.ELA.Arbiter/commit/1144e145a17132e3a96184e1faf07359f44be55d))

