# Desktop Commander Bounded Process Overlay V1 Plan

1. Record a focused red proving the WorkBridge admission module is absent.
2. Add a four-slot FIFO ProcessAdmissionGate and focused 4+1 test.
3. Wrap the pinned terminal manager executeCommand lifetime with the gate.
4. Teach Install-DesktopCommanderDuplicate.ps1 to apply the overlay before build and to require the focused test.
5. Build into a temporary install root and run the complete upstream test suite.
6. Preserve the current live duplicate tree as rollback evidence, install the qualified overlay build, restart only the duplicate/tunnel path as required, and prove four concurrent start_process calls overlap while a fifth waits.
7. Record source head, upstream pin, manifest/hash evidence, test results, and live proof before integration.