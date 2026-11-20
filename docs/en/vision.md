<!-- docs-pair: vision; revision: 1 -->

[English](vision.md) | [Español](../es/vision.md) | [Index](README.md)

# Vision and boundaries

HiveScope aims to make privately controlled, community-operated storage possible
without depending on a central storage provider. Participants exchange explicitly
limited resources while keeping ownership of their files and keys.

## First useful service

Synchronize a private folder between the owner's authorized devices. Approved
community providers retain encrypted fragments and help restore availability.
An approved provider does not automatically become a reader of the folder.

The initial network uses bilateral agreements. Each provider decides whom to
serve, for how much physical capacity, for how long and at which transfer limits.
There is no network-wide balance, token, blockchain or universal exchange rate.
Approval is a local decision rather than an account in a central registry.

Some community nodes are expected to be regularly online. Offline editing remains
possible, but remote recovery needs enough reachable fragments and metadata.
The alpha will not advertise a durability or availability SLA.

## Participant control

Sharing is opt-in. A participant must configure the storage directory, quota,
upload/download limits and schedule before accepting data. The design includes
manual pause and battery-aware pause. Detecting games or video calls is later work.
Departing providers should allow migration during their agreement's remaining
term; abrupt failures are still part of the threat and reliability model.

## Meaning of free

The goal is no mandatory service fee. Hardware, electricity, connectivity,
maintenance and relay traffic still have costs paid by participants or donors.
An offer of disk space does not guarantee access to scarce GPU capacity.
The project does not promise unlimited storage, bandwidth or inference.

## Later community AI

Start with existing open models executed in full by capable providers, routing
requests among them under separate compute agreements. Evaluate distributed
inference across model partitions after measuring that service. Model training
requires an additional data, licensing, evaluation and compute plan.

Storage encryption does not make inference private: a remote compute provider
may see the inputs it processes. Model licenses must be evaluated individually;
open weights are not automatically an open-source license.

## Outside the first implementation

Public download links, multi-user editing, mobile clients, transparent cloud-only
files, global credits, permissionless admission, model training and anonymous
networking are outside the first storage implementation. The initial architecture
targets Windows, Linux and macOS with a Go daemon and CLI; a GUI follows later.
