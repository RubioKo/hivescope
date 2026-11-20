<!-- docs-pair: resource-offers; revision: 4 -->

English | [Español](../es/resource-offers.md) | [Index](README.md)

# Offer my computer

HiveScope's proposed desktop flow starts with two independent choices. Ana can
request a copy or a compute session without lending any hardware. Luis can offer
his computer to the open network by activating a local resource profile once.
Their devices can then reach an agreement without an invitation or a conversation.

## What Luis offers

Luis opens **Offer my computer**, selects a service and reviews the limits. For
example, he could lend up to 20 GB for encrypted storage in the evenings and
keep his GPU for himself. Before activating the profile, he sees his schedule
and time zone, any requirement to be plugged in, the space already committed
and the resources he will keep. Leaving the draft shares nothing.

| Service | What Luis chooses | What the agreement covers |
| --- | --- | --- |
| Encrypted storage | A directory, total disk quota, space to keep free, transfer limits and hours. | The host holds ciphertext. Reservations, metadata and temporary files count toward the physical quota. The folder owner chooses which devices may read the files. |
| Private compute | CPU, RAM, optional GPU, bandwidth, number of sessions and available hours. | The requester chooses and runs software inside a verified private environment. Luis controls the physical machine and enforces the resource limits. |
| Hosting a service or game | Capacity for the session, persistent disk, bandwidth and availability. | The requester controls the service and its state. A standby must satisfy the session's protection requirements and have a verified save to take over. |

The directory advertisement includes the service, the node's contact information
and an expiry. File names, session contents and Luis's personal schedule stay
out of the directory. Participants can still observe operational details such
as connection times and transfer sizes. [Security model](security.md)

## How the machines reach an agreement

1. Ana chooses a folder to protect or describes the resources her session needs.
   Her node queries a community directory. If it is unavailable, the node can
   try another. An advertisement leads to a candidate.
2. The devices authenticate each other. Luis's node checks the profile, current
   time, free capacity on the offered device and outstanding commitments. It
   grants a time-limited reservation only if the request fits both the total
   host budget and the limits for that requester. Two requests for the last
   available slot can receive at most one reservation.
3. Ana's device uses the reservation for the agreed operation. A retry with the
   same request identifier retrieves its existing state and counts once. An
   unused reservation expires and releases the capacity. An accepted storage
   agreement has its own retention and retrieval terms.
4. If no suitable host is available, Ana sees what she is waiting for and can
   cancel. Her node can try another candidate under the same conditions. A
   missing result after a disconnect stays unconfirmed until the devices
   reconcile the request.

For private compute, Ana's device verifies fresh evidence for the confidential
virtual machine and its session key before sending secrets or a private
workload. If her program uses a GPU, verification must cover the GPU and the
protected path to it as well. When a candidate fails verification, Ana keeps
her keys and the request waits for a compatible host. Once verified, she can
load her own tools, code and data into the session. A later session on a
different host requires its own verification. [Architecture](architecture.md)

## Pausing or shutting down

**Pause** immediately stops new reservations. Luis can see the work already
admitted and what the next shutdown would interrupt. Only Luis can enlarge his
profile. His computer remains under his control even during a failed handoff.

| Work in progress | What the participants need to see |
| --- | --- |
| Stored copies | A powered-off host cannot serve its fragments. Retention, migration and expiry follow the storage agreement. The owner sees which replacements have been confirmed. |
| Compute session | At the end of the agreed time or if Luis stops the session, Ana sees the last verified result or checkpoint, if her program saved one. The host releases the session's CPU, RAM and GPU. |
| Service or game | Show whether a compatible standby can restore the most recent verified state. Only one host may accept updates. If the handoff cannot finish, report the interruption so clients can reconnect when service resumes. |

After an unexpected outage, a lost connection is the first observable event.
The requester checks the last receipt, result or save before reporting recovery.
[Storage and repair](storage.md)

## How to test the design

- Check the request and offer descriptions against the local profile. The requester
  chooses the software, the host sets physical limits and the requester retains
  the keys while suitable hardware is unavailable.
- Start with offering turned off. Activate a disk-only profile, request CPU, then
  request disk outside its schedule. Check that neither request occupies capacity.
- Interrupt a transfer after the host has stored it but before the requester
  sees its receipt. Check that a retry produces one stored object and one charge
  against the quota. Pause new work and verify existing retention commitments.
- Try a private session with missing or stale evidence and another with an
  unverified GPU. Check that session keys and private inputs stay on the
  requesting device, including after retry. On compatible hardware, run a
  requester-chosen program and recover its result.
