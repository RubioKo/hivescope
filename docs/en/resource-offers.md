<!-- docs-pair: resource-offers; revision: 2 -->

English | [Español](../es/resource-offers.md) | [Index](README.md)

# Offer my computer

This is a proposed desktop flow. When Luis selects **Offer my computer**, he
opens a local draft. All three activities start unselected. **Continue** becomes
available once he chooses one. He can leave without sharing any resources. The first
planned implementation covers storage through the command line. Transcription of
public catalog recordings and game hosting illustrate later services.

## What the person lending the computer chooses

1. **Service and computer.** Luis chooses his computer and a specific activity.
   He can create separate offers to store encrypted copies, transcribe public catalog audio
   with an approved tool or host an approved game server.
2. **Who can use it.** Luis chooses an approved contact for this agreement. He can
   make a separate offer to another person within his computer's total capacity. For a game, the
   summary names the person responsible for the world. Joining as a player
   grants only in-game permissions.
3. **How much and when.** Luis sets resource limits, a schedule with its time
   zone, how long the offer is valid and whether the computer must be plugged
   in. He sees how much capacity is already committed and how much he will have
   left for himself. If overlapping offers exceed a local limit, he must adjust
   one before sending the offer.

| Service | Limits Luis reviews | Data his computer can access |
| --- | --- | --- |
| Encrypted copy | Destination directory, disk quota, free space to keep in reserve, and upload and download limits. Temporary files count toward the quota. | Encrypted fragments and transfer sizes. Reading keys stay on the devices authorized by the folder owner. |
| Public catalog transcription | Approved tool, CPU, memory, optional GPU use and the number of jobs that can run at once. | Luis receives a public catalog identifier. He can hear the published recording it refers to and read the transcript. |
| Game server | Game and version, CPU, memory, disk, network connection, schedule and a possible backup host. | The world data the server processes. A handoff requires authorization and a valid save. |

## What the confirmation button authorizes

The summary shows the contact, the service, its limits, the data the computer
will receive and what will happen if Luis interrupts the service. He can select
**Go back and edit** or **Send offer to Ana**. For example, he could offer to
transcribe one public catalog recording at a time for Ana, using up to 2 CPU cores and 4 GB of
memory, with the GPU off, from 18:00 to 21:00 in his time zone and only while
plugged in. Luis chose those values for this example offer.

Once sent, the offer is **awaiting acceptance**. Luis can withdraw it while pending.
If it expires before Ana accepts, a new offer is needed. Once Ana accepts the same
terms, the agreement authorizes her requests that match the service, schedule and
limits. In this example, Ana chooses a public catalog entry and confirms the
job. The only input she can submit is its identifier. Luis retrieves the published
audio directly. Changing the provider for that job requires Ana's approval.

If Ana opens her own interview to transcribe it, HiveScope would run the task
on her device. Her recording and transcript stay local, including when resources
are insufficient or the job fails. Accepting Luis's offer cannot redirect a
private recording to his computer.

Adding recipients, increasing resources or extending permissions requires
review and acceptance of the new terms. Luis can pause new requests immediately.
Outside the agreed hours, the computer stops accepting jobs. A running job may
use resources beyond that window only if Luis agrees to extend it.

## If Luis needs to shut down his computer

**Prepare to shut down** pauses new requests and shows what is still in use.
Luis can review the consequences and decide whether to wait or stop the work.
**Go back** lets him continue monitoring the current job. To accept new requests
again, he selects **Resume** during the agreed hours.
He can still shut down from his operating system while a handoff is pending.

| What is still in use | What the screen should explain |
| --- | --- |
| Encrypted copies | While the computer is off, it cannot serve its fragments. It keeps local data according to the agreement. If Luis plans to withdraw, he can see which copies have been confirmed as migrated and which are still pending. |
| A public catalog transcription | Luis can wait for it to finish or select **Stop job**. If he stops it, Ana sees the interruption once her device receives confirmation. The screen shows recoverable progress only when the tool has saved and verified it. |
| A game world | Before stopping the server, Luis sees the last valid save and whether a handoff is confirmed. Players may need to reconnect. Without a backup host, the world goes offline. Only one host can accept changes to the world at a time. |

If power is lost, other devices can report only that they have lost contact.
When they reconnect, they must check the last valid result, save or receipt
before reporting a recovery. Withdrawing a disk offer leaves the retention,
migration and expiry obligations of the [storage agreement](storage.md) in
effect. How to preserve progress on a transcription remains an open question.

## How to test the decision

- Leave the draft and check that it did not enable any service.
- Agree to provide only disk space for Ana, then try to request a transcription
  under that agreement. The request must remain unauthorized.
- With a computing agreement in place, try to submit a local recording, private
  instructions or an unknown catalog identifier. Each request must be rejected
  before any input leaves the requester. A valid public catalog entry should run.
- Interrupt a private transcription on Ana's device and retry with a volunteer
  host available. Verify that her audio and transcript stay on her device.
- Show the summary to someone lending their computer for the first time. Before
  they send it, ask them to point out who can use the computer, the limits and
  what happens if they shut it down.
- Try a planned shutdown and an unexpected outage during a job. Check what each
  person sees when their device receives confirmation and when it loses contact.
