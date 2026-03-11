#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use serde::Serialize;

#[derive(Serialize)]
struct Diagnostics {
    runtime: String,
    app: String,
    timestamp: String,
}

#[tauri::command]
fn collect_diagnostics() -> Diagnostics {
    Diagnostics {
        runtime: "tauri".to_string(),
        app: "opener-netdoor-manager".to_string(),
        timestamp: chrono::Utc::now().to_rfc3339(),
    }
}

fn main() {
    tauri::Builder::default()
        .invoke_handler(tauri::generate_handler![collect_diagnostics])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
