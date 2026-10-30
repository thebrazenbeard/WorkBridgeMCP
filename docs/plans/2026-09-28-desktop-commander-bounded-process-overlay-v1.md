# Desktop Commander Bounded Process Overlay V1 Plan

1. Record a focused red proving the WorkBridge admission module is absent.
2. Add a four-slot FIFO ProcessAdmissionGate and focused 4+1 test.
3. Acquire before spawning a direct child and hold its slot until child exit,
   close, or failed spawn. The initial `executeCommand` response can return
   promptly with a PID while its child continues running; that response must
   not release the slot. Descendants surviving the direct shell PID are not
   bounded by this gate.
4. Teach Install-DesktopCommanderDuplicate.ps1 to apply the overlay before build and to require the focused test.
5. Build into a temporary install root and run the complete upstream test suite.
6. Preserve the current live duplicate tree as rollback evidence, install the qualified overlay build, restart only the duplicate/tunnel path as required, and prove four live direct `start_process` PIDs overlap while a fifth waits until one exits.
7. Record source head, upstream pin, manifest/hash evidence, test results, and live proof before integration.
