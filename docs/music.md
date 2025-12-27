# Music

## Typical structure

The dynamic music tracks have 12 channels, made up of 6 stereo pairs:

- Channels 00-01: [Digital silence]
- Channels 02-03: Lead guitars & melody
- Channels 04-05: Bass guitar
- Channels 06-07: Drum/primary percussion
- Channels 08-09: Heavy/native backing percussion
- Channels 10-11: Orchestral

## Music states

For tracks that have the above structure — that is, multiple stems — it has it so it can be dynamically mixed.

The following are the dynamic states I've observed in gameplay and the channels that play during that state:

| State              | Lead melody     | Bass melody     | Primary Percussion   | Backing percussion   | Orchestra   |
|--------------------|-----------------|-----------------|----------------------|----------------------|-------------|
| Standard battle    | ✅              | ✅              | ✅                   | ✅                   | ❌          |
| Paused             | ✅              | ✅              | ❌                   | ❌                   | ❌          |
| Activate Ultimate  | ❌              | ✅              | ✅                   | ❌                   | ❌          |
| Superior (Winning) | ✅              | ✅              | ✅                   | ✅                   | ✅          |

TODO: Confirm these are accurate, what other states exist
