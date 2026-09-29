# DS216 Boundary for the Desktop Commander App

Status: PLATFORM SEPARATION / DO NOT FOLD INTO V1

## Decision

Do not make the DS216 a prerequisite for the first WorkBridge Desktop Commander ChatGPT app.

The first app is Lappy-only.

## Why

The exact Desktop Commander duplicate has an upstream Node requirement of Node >=18 and includes platform-sensitive/native dependencies such as ripgrep and sharp.

The DS216 is an ARMv7 DSM appliance. Earlier qualification work established a viable static Go ARMv7 WorkBridge build path. That is materially simpler and more credible for the NAS than assuming the entire Desktop Commander Node/native dependency stack will be durable on that appliance.

## Current DS216 direction

Use bounded native WorkBridge as the NAS-side candidate:

```text
ChatGPT / future broker
  -> VeraMesh transport
  -> native ARMv7 WorkBridge
  -> admitted DS216 roots / explicit process grants
```

This is intentionally not semantically identical to Desktop Commander.

## Future convergence options

After Lappy V1 passes, a multi-device product could choose one of four models:

1. expose different per-device MCP servers/apps;
2. add a broker that presents device selection plus device-native tools;
3. define a common lowest-common-denominator contract;
4. add a compatibility facade that implements Desktop-Commander-like operations over native WorkBridge where behavior can be proven equivalent.

No option is selected by V1.

## Rule

Do not claim DS216 parity merely because both devices can read/write files or run commands. Process/session semantics, document handling, search behavior, configuration, and tool schemas are separate contracts.

## SSH

SSH remains an independent admin/recovery plane for the DS216 and Lappy. It is not the normal WorkBridge Desktop Commander transport.
