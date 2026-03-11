# Stage 12 mobile skeleton

This Flutter module is intentionally minimal and keeps API/session/state seams ready.

Current validated structure:
- `pubspec.yaml`
- `analysis_options.yaml`
- `lib/main.dart`
- `lib/src/*` screens/services/state

Platform folders (`android`, `ios`, `macos`, `linux`, `windows`, `web`) are not committed in this pass because Flutter tooling is unavailable in the current environment.
Generate them with:

```bash
cd apps/mobile-client
flutter create .
```
