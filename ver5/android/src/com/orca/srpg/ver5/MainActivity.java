package com.orca.srpg.ver5;

import android.app.Activity;
import android.app.AlertDialog;
import android.os.Bundle;
import android.util.Log;
import android.view.KeyEvent;
import android.view.View;
import android.view.WindowManager;
import android.widget.TextView;
import com.orca.srpg.mobile.Mobile;
import com.orca.srpg.mobile.EbitenView;
import java.io.File;
import java.io.FileInputStream;
import java.io.FileOutputStream;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.util.zip.ZipEntry;
import java.util.zip.ZipInputStream;

public final class MainActivity extends Activity {
    private EbitenView gameView;
    private boolean resumed;

    @Override public void onCreate(Bundle state) {
        super.onCreate(state);
        getWindow().addFlags(WindowManager.LayoutParams.FLAG_KEEP_SCREEN_ON);
        immersive();
        TextView loading = new TextView(this);
        loading.setText("도원결의를 불러오는 중…");
        loading.setTextSize(24);
        loading.setGravity(android.view.Gravity.CENTER);
        setContentView(loading);
        new Thread(() -> {
            try {
                unpackAssets();
                Mobile.start(getFilesDir().getAbsolutePath());
                runOnUiThread(() -> {
                    if (isFinishing() || isDestroyed()) return;
                    gameView = new EbitenView(this) {
                        @Override protected void onErrorOnGameUpdate(Exception error) {
                            showError(error);
                        }
                    };
                    setContentView(gameView);
                    if (resumed) gameView.resumeGame();
                    else gameView.suspendGame();
                });
            } catch (Exception error) {
                runOnUiThread(() -> showError(error));
            }
        }, "srpg-load").start();
    }

    private void unpackAssets() throws IOException {
        String revision;
        try (java.io.InputStream in = getAssets().open("runtime.sha256")) {
            byte[] bytes = new byte[64];
            int n = 0, count;
            while (n < bytes.length && (count = in.read(bytes, n, bytes.length - n)) > 0) n += count;
            revision = new String(bytes, 0, n, StandardCharsets.US_ASCII);
        }
        File root = new File(getFilesDir(), "runtime");
        File marker = new File(root, ".revision");
        if (marker.isFile()) {
            try (FileInputStream in = new FileInputStream(marker)) {
                byte[] bytes = new byte[64];
                if (in.read(bytes) == 64 && revision.equals(new String(bytes, StandardCharsets.US_ASCII))) return;
            }
        }
        if (!root.isDirectory() && !root.mkdirs()) throw new IOException("Cannot create runtime directory");
        String prefix = root.getCanonicalPath() + File.separator;
        try (ZipInputStream zip = new ZipInputStream(getAssets().open("runtime.zip"))) {
            ZipEntry entry;
            byte[] buffer = new byte[65536];
            while ((entry = zip.getNextEntry()) != null) {
                File target = new File(root, entry.getName());
                if (!target.getCanonicalPath().startsWith(prefix)) throw new IOException("Invalid asset path");
                File parent = target.getParentFile();
                if (!parent.isDirectory() && !parent.mkdirs()) throw new IOException("Cannot create asset directory");
                if (!entry.isDirectory()) {
                    try (FileOutputStream out = new FileOutputStream(target)) {
                        int count;
                        while ((count = zip.read(buffer)) != -1) out.write(buffer, 0, count);
                    }
                }
                zip.closeEntry();
            }
        }
        try (FileOutputStream out = new FileOutputStream(marker)) {
            out.write(revision.getBytes(StandardCharsets.US_ASCII));
        }
    }

    private void showError(Exception error) {
        Log.e("SRPG", "Game initialization/update failed", error);
        if (isFinishing() || isDestroyed()) return;
        new AlertDialog.Builder(this).setTitle("실행 오류").setMessage(error.toString())
            .setPositiveButton("닫기", (dialog, which) -> finish()).show();
    }

    private void immersive() {
        getWindow().getDecorView().setSystemUiVisibility(View.SYSTEM_UI_FLAG_FULLSCREEN
            | View.SYSTEM_UI_FLAG_HIDE_NAVIGATION | View.SYSTEM_UI_FLAG_IMMERSIVE_STICKY
            | View.SYSTEM_UI_FLAG_LAYOUT_FULLSCREEN | View.SYSTEM_UI_FLAG_LAYOUT_HIDE_NAVIGATION
            | View.SYSTEM_UI_FLAG_LAYOUT_STABLE);
    }

    @Override public void onWindowFocusChanged(boolean focused) {
        super.onWindowFocusChanged(focused);
        if (focused) immersive();
    }

    @Override protected void onPause() {
        resumed = false;
        if (gameView != null) gameView.suspendGame();
        super.onPause();
    }

    @Override protected void onResume() {
        super.onResume();
        resumed = true;
        if (gameView != null) gameView.resumeGame();
        immersive();
    }

    @Override public void onBackPressed() {
        if (gameView != null) Mobile.back();
        else super.onBackPressed();
    }

    @Override public boolean dispatchKeyEvent(KeyEvent event) {
        // EbitenView consumes key events, including Back, before Activity handles them.
        if (event.getKeyCode() == KeyEvent.KEYCODE_BACK && gameView != null) {
            if (event.getAction() == KeyEvent.ACTION_DOWN && event.getRepeatCount() == 0) Mobile.back();
            return true;
        }
        return super.dispatchKeyEvent(event);
    }
}
