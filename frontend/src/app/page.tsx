"use client";

import { useEffect, useState, useCallback, useMemo } from "react";
import { useDropzone } from "react-dropzone";
import { FileItem, FileStatus, FileMetadata } from "../types";

type UpdateResult = {
  available: boolean;
  version: string;
  release_notes: string;
  download_url: string;
  error?: string;
};
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
// import { cn } from "@/lib/utils"; // Not used currently
import { Loader2, CheckCircle2, XCircle, FileVideo, Plus, Zap, Trash2, Play, X, FolderPlus, Moon, Sun, Download, RefreshCw } from "lucide-react";

// Mock Wails Runtime for Dev/Build
const getWailsApp = () => {
  if (typeof window !== "undefined" && (window as any).go?.bridge?.App) {
    return (window as any).go.bridge.App;
  }
  return null;
};

// Mock Wails Runtime Events
const getWailsEvents = () => {
  if (typeof window !== "undefined" && (window as any).runtime) {
    return (window as any).runtime;
  }
  return null;
}

const formatSize = (bytes: number) => {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
};

const formatTime = (seconds: number) => {
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = Math.floor(seconds % 60);
  return h > 0 ? `${h}:${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}` : `${m}:${s.toString().padStart(2, '0')}`;
};

export default function Home() {
  const [files, setFiles] = useState<FileItem[]>([]);
  const [isWailsReady, setIsWailsReady] = useState(false);
  const [playingFile, setPlayingFile] = useState<FileItem | null>(null);
  const [theme, setTheme] = useState<'light' | 'dark'>('light');
  const [appVersion, setAppVersion] = useState("0.0.0");
  const [updateInfo, setUpdateInfo] = useState<UpdateResult | null>(null);
  const [isUpdating, setIsUpdating] = useState(false);
  const [updateReady, setUpdateReady] = useState(false);
  const [wailsConnected, setWailsConnected] = useState(false);

  useEffect(() => {
    // Check for updates on mount
    if (getWailsApp()) {
      checkForUpdates();
    }
  }, []); // Run once on mount

  const checkForUpdates = async () => {
    // Default to raw github url
    const updateUrl = "https://raw.githubusercontent.com/billytoe/mp4-optimizer/main/latest.json";
    const app = getWailsApp();
    if (!app) return;

    try {
      const result = await app.CheckForUpdates(updateUrl);
      if (result && result.available) {
        setUpdateInfo(result);
        // Automatically start download
        performUpdate(result.download_url);
      }
    } catch (e) {
      console.error("Failed to check updates:", e);
    }
  };

  const performUpdate = async (url: string) => {
    setIsUpdating(true);
    const app = getWailsApp();
    try {
      await app.InstallUpdate(url);
      setIsUpdating(false);
      setUpdateReady(true);
    } catch (e) {
      console.error("Update failed:", e);
      setIsUpdating(false);
      // Optional: show error state
    }
  };

  const handleRestart = () => {
    const runtime = getWailsEvents();
    if (runtime) {
      runtime.Quit();
    } else {
      // Fallback for dev?
      window.location.reload();
    }
  };
  useEffect(() => {
    const savedTheme = localStorage.getItem('theme') as 'light' | 'dark';
    if (savedTheme) {
      setTheme(savedTheme);
      document.documentElement.classList.toggle('dark', savedTheme === 'dark');
    } else {
      // Default to light as requested
      setTheme('light');
      document.documentElement.classList.remove('dark');
    }
  }, []);

  const toggleTheme = () => {
    const newTheme = theme === 'light' ? 'dark' : 'light';
    setTheme(newTheme);
    localStorage.setItem('theme', newTheme);
    document.documentElement.classList.toggle('dark', newTheme === 'dark');
  };

  useEffect(() => {
    // Robust Wails Runtime Loader
    let intervalId: NodeJS.Timeout;

    const bindWails = () => {
      const runtime = getWailsEvents();
      if (runtime) {
        console.log("Wails runtime found, binding events.");

        // Prevent double binding if we already bound?
        // Wails EventsOn returns a cleanup function usually? No, Wails v2 JS runtime doesn't return unbind for EventsOn easily in types.
        // But 'EventsOff' exists. We should clear first to be safe.
        runtime.EventsOff("files-dropped");

        runtime.EventsOn("files-dropped", (paths: string[]) => {
          console.log("Event: files-dropped received", paths);
          alert("Debug: Files Dropped (Success) -> " + JSON.stringify(paths));
          if (paths && paths.length > 0) {
            addFiles(paths);
          }
        });

        // Get Version (Moved inside to ensure Wails is ready)
        if (getWailsApp()) {
          getWailsApp().GetAppVersion().then((v: string) => {
            if (v) {
              setAppVersion(v);
              // Check for updates after we have the version
              checkForUpdates();
            }
          });
        }

        setWailsConnected(true);
        setIsWailsReady(true);

        if (intervalId) clearInterval(intervalId);
        return true;
      }
      return false;
    };

    // Try immediately
    if (!bindWails()) {
      // Poll every 100ms for up to 5 seconds
      let attempts = 0;
      intervalId = setInterval(() => {
        attempts++;
        if (bindWails() || attempts > 50) {
          if (attempts > 50) console.error("Wails Connection Timed Out");
          clearInterval(intervalId);
        }
      }, 100);
    }

    return () => {
      if (intervalId) clearInterval(intervalId);
    };
  }, []);


  const addFiles = useCallback(async (newPaths: string[]) => {
    // Expand paths (handle directories and invalid paths via backend)
    let processedPaths = newPaths;
    const app = getWailsApp();

    if (app) {
      try {
        console.log("Expanding paths:", newPaths);
        // Use the new backend method to valid/expand paths
        processedPaths = await app.ExpandPaths(newPaths);
        console.log("Expanded paths:", processedPaths);
      } catch (e) {
        console.error("Failed to expand paths:", e);
        // Fallback to original paths if expansion fails (unlikely)
      }
    } else {
      // Dev mode fallback: filter empty strings
      processedPaths = newPaths.filter(p => !!p);
    }

    if (!processedPaths || processedPaths.length === 0) return;

    const newItems: FileItem[] = processedPaths.map((path) => ({
      id: path, // Use path as ID for simplicity
      path,
      name: path.split(/[/\\]/).pop() || path,
      size: 0, // We could get size via Wails, for now 0
      status: "pending" as FileStatus,
    }));

    // Filter duplicates
    setFiles((prev) => {
      // Use map to keep latest added (though we filter new ones)
      // Actually we want to filter OUT newItems that are already in prev
      const existingPaths = new Set(prev.map((f) => f.path));
      const filtered = newItems.filter((f) => !existingPaths.has(f.path));
      return [...prev, ...filtered];
    });

    // scan immediately
    newItems.forEach(async (item) => {
      // Check if it's already in the list to avoid double scanning (if setFiles filtered it out)
      // BUT `newItems` contains everything.
      // We should only scan what we actually ADDED.
      // However, `setFiles` is async, so we can't easily know here.
      // It's acceptable to re-scan or scan duplicate requests (debounce handled by status check usually)

      // Better: filter newItems against CURRENT files state ref?
      // For now, scanning everything passed is safe enough.
      scanFile(item.path);
      fetchMetadata(item.path);
    });
  }, []); // Remove `files` dep to avoid loop, use functional update

  const scanFile = async (path: string) => {
    // Only update if not already scanning?
    updateFileStatus(path, "scanning");
    const app = getWailsApp();
    if (!app) {
      // Mock for dev
      setTimeout(() => {
        updateFileStatus(path, Math.random() > 0.5 ? "optimized" : "unoptimized");
      }, 500);
      return;
    }

    try {
      const isOptimized = await app.CheckFile(path);
      updateFileStatus(path, isOptimized ? "optimized" : "unoptimized");
    } catch (e: any) {
      updateFileStatus(path, "error", e.toString());
    }
  };

  const updateFileStatus = (path: string, status: FileStatus, message?: string) => {
    setFiles((prev) =>
      prev.map((f) => (f.path === path ? { ...f, status, message } : f))
    );
  };

  const updateFileMetadata = (path: string, metadata: FileMetadata) => {
    setFiles((prev) =>
      prev.map((f) => (f.path === path ? { ...f, metadata, size: metadata.size } : f))
    );
  };

  const fetchMetadata = async (path: string) => {
    const app = getWailsApp();
    if (!app) return;
    try {
      const meta = await app.GetFileMetadata(path);
      if (meta) {
        updateFileMetadata(path, {
          ...meta,
          // Ensure modified is a string if it comes as string from JSON, or Date object?
          // Go 'time.Time' usually marshals to RFC3339 string in JSON.
          modified: meta.modified
        });
      }
    } catch (e) {
      console.error("Failed to get metadata", e);
    }
  }

  const optimizeFile = async (path: string) => {
    // FIX: Unload video if it's currently playing to prevent file locking on Windows
    if (playingFile && playingFile.path === path) {
      setPlayingFile(null);
      // Wait for React re-render and browser to release handle
      await new Promise(resolve => setTimeout(resolve, 500));
    }

    updateFileStatus(path, "scanning"); // use scanning spinner
    const app = getWailsApp();
    if (!app) {
      setTimeout(() => {
        updateFileStatus(path, "optimized");
      }, 1000);
      return;
    }

    try {
      await app.OptimizeFile(path);
      updateFileStatus(path, "optimized");
    } catch (e: any) {
      updateFileStatus(path, "error", e.toString());
    }
  };

  const handleOpenDirectory = async () => {
    const app = getWailsApp();
    if (app) {
      try {
        const path = await app.SelectDirectory();
        if (path) {
          addFiles([path]);
        }
      } catch (e) {
        console.error(e);
      }
    }
  };

  const handleOpenFiles = async () => {
    const app = getWailsApp();
    if (app) {
      try {
        const paths = await app.SelectFiles();
        if (paths && paths.length > 0) {
          addFiles(paths);
        }
      } catch (e) {
        console.error(e);
      }
    } else {
      alert("Wails OpenDialog not available in browser");
    }
  };

  const handleOptimizeAll = async () => {
    const unoptimizedFiles = files.filter(f => f.status === 'unoptimized');
    if (unoptimizedFiles.length === 0) return;

    // Concurrently optimize? Or sequential?
    // Sequential is safer for disk I/O, concurrent is faster.
    // Go backend handles rename, so concurrent is okayish but might saturate I/O.
    // Let's do parallel, it's user friendly.
    unoptimizedFiles.forEach(f => optimizeFile(f.path));
  }

  // Count unoptimized files
  const unoptimizedCount = useMemo(() => files.filter(f => f.status === 'unoptimized').length, [files]);

  // Dropzone for web fallback (and visual overlay trigger)
  // We disable native drop handling effectively by not using its file objects if in Wails
  // But strictly speaking, react-dropzone might steal the event.
  // We should test. Usually Wails OnFileDrop fires BEFORE WebView gets it if `DragAndDrop` is handled?
  // Wails 2 runtime drop is usually overlaying.
  // Let's keep dropzone for "Click to open" or just visuals.
  // Drag-and-Drop Strategy:
  // In Wails, we want the "native" window drop event (runtime.OnFileDrop) to work.
  // We strictly disable react-dropzone's interference in Wails mode.
  // We also prevent default browser behaviors (navigation) globally.
  // Drag-and-Drop Strategy:
  // 1. We enable react-dropzone always to provide visual feedback and standard behavior.
  // 2. On Windows (WebView2), `acceptedFiles` usually contains the full absolute path in `f.path`.
  // 3. On macOS (WebKit), `f.path` is often empty or just the filename (security restriction).
  // 4. Wails Runtime also emits "files-dropped" with absolute paths on both platforms (if configured).
  //
  // FIX for Windows:
  // We MUST allow react-dropzone to handle the event to prevent "no reaction".
  // In `onDrop`, we check if we have absolute paths.
  // - If YES (Windows): We use them immediately.
  // - If NO (Mac): We ignore them here and rely on the "files-dropped" runtime event.
  // This ensures both platforms work reliably.

  // Restore global event listeners to ensure WebView2 accepts the drag (prevents 🚫 cursor)
  // Hybrid Drag Config:
  // 1. We enable WebView drop (DisableWebViewDrop: false).
  // 2. We add global dragover listener to prevent "No Drop" cursor.
  // 3. We listen to BOTH react-dropzone onDrop AND Wails files-dropped event.
  // 4. We deduplicate in addFiles.

  useEffect(() => {
    // Always attach this to ensure Windows allows the drag
    const preventDefault = (e: Event) => {
      e.preventDefault();
    };

    window.addEventListener("dragover", preventDefault);
    window.addEventListener("drop", preventDefault);

    return () => {
      window.removeEventListener("dragover", preventDefault);
      window.removeEventListener("drop", preventDefault);
    };
  }, []); // Run once on mount

  const { getRootProps, getInputProps, isDragActive } = useDropzone({
    onDrop: (acceptedFiles) => {
      // Hybrid: Try to get paths from dropzone first
      const paths: string[] = [];

      // Debug: see what we got
      // alert("Frontend Drop Raw: " + JSON.stringify(acceptedFiles.map(f => ({name: f.name, path: (f as any).path}))));

      // Try to get absolute paths (Windows WebView2)
      acceptedFiles.forEach((f: any) => {
        if (f.path) paths.push(f.path);
        else if (f.name) paths.push(f.name); // Fallback
      });

      if (paths.length > 0) {
        console.log("React-Dropzone received paths:", paths);
        // Only alert if we found something, to compare with backend
        // alert("Debug: React-Dropzone Found -> " + JSON.stringify(paths));

        // Note: On Windows with DisableWebViewDrop: true, this frontend handler MIGHT NOT fire at all.
        // We rely on the Wails 'files-dropped' event (handled in bindWails) for the absolute path.
        // But if it does fire (e.g. Mac), we can use it.
        addFiles(paths);
      }
    },
    noClick: true,
    noKeyboard: true,
    noDrag: false, // Enable react-dropzone handling
  });

  return (
    <div
      className="flex flex-col h-screen bg-background text-foreground font-sans p-6 transition-colors duration-300"
      {...getRootProps()}
    >

      {/* Overlay Drop Zone Visuals - Only show if dragging and NOT in Wails mode (since we disable dropzone logic there) 
          OR we manually detect drag? 
          Actually if `noDrag` is true, `isDragActive` will never be true.
          So we lose the visual feedback if we disable react-dropzone.
          
          Compromise: Enable react-dropzone but don't PROCESS the files in onDrop if Wails is ready.
          But if react-dropzone calls preventDefault/stopPropagation, Wails might not see the file drop.
          
          If we use `noDrag={false}` (default) -> react-dropzone handles it -> Wails might miss it.
          Check Wails docs/issues: Wails OnFileDrop usually works even if WebView handles it?
          Actually, often WebView swallows it.
          
          Safest bet: Disable react-dropzone's drop handling in Wails, but how to keep visual?
          We can listen to window 'dragover' ourselves for visual.
          
          For now, let's keep `noDrag={false}` (default).
          And in `onDrop`, we simply return if `isWailsReady`.
          Question is: does `react-dropzone` stop propagation? Yes it does.
          
          If specific Wails version requires event bubbling, this might be an issue.
          Ref: Wails v2 usually intercepts at native window level before WebView.
          So it should be fine.
      */}
      <input {...getInputProps()} />
      {/* Visual only */}
      {isDragActive && (
        <div className="absolute inset-0 bg-blue-500/20 z-50 flex items-center justify-center backdrop-blur-sm border-4 border-blue-500 border-dashed m-4 rounded-xl pointer-events-none">
          <p className="text-4xl font-bold text-blue-400">请释放 MP4 文件到此处</p>
        </div>
      )}

      <header className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-3xl font-extrabold tracking-tight bg-gradient-to-r from-blue-500 to-emerald-500 bg-clip-text text-transparent">
            FastStart 视频优化
          </h1>
          <p className="text-muted-foreground mt-1">检测与优化 MP4 流媒体播放性能</p>
        </div>
        <div className="flex gap-4 items-center">
          {/* Auto-Update Status Indicators */}
          {isUpdating && (
            <Badge className="bg-emerald-600/10 text-emerald-600 border-emerald-500/20 px-3 py-1 h-9 gap-2">
              <Loader2 className="w-4 h-4 animate-spin" />
              Downloading...
            </Badge>
          )}
          {updateReady && (
            <Button size="sm" onClick={handleRestart} className="bg-emerald-600 hover:bg-emerald-500 text-white h-9 shadow-lg animate-pulse gap-2">
              <RefreshCw className="w-4 h-4" />
              Restart to Update
            </Button>
          )}

          <Button variant="ghost" size="icon" onClick={toggleTheme} className="text-muted-foreground hover:text-foreground">
            {theme === 'dark' ? <Sun className="w-5 h-5" /> : <Moon className="w-5 h-5" />}
          </Button>
          <Badge variant="outline" className="h-10 px-4 text-muted-foreground border-border bg-muted/50">
            共 {files.length} 个文件
          </Badge>
          <Button variant="outline" onClick={() => setFiles([])} disabled={files.length === 0}>
            <Trash2 className="w-4 h-4 mr-2" />
            清空
          </Button>

          {/* Batch Optimization Button */}
          <Button
            onClick={handleOptimizeAll}
            disabled={unoptimizedCount === 0}
            className={unoptimizedCount > 0 ? "bg-emerald-600 hover:bg-emerald-500 text-white" : "bg-slate-800 text-slate-500"}
          >
            <Zap className="w-4 h-4 mr-2" />
            全部优化 ({unoptimizedCount})
          </Button>

          <Button onClick={handleOpenFiles} size="lg" className="bg-blue-600 hover:bg-blue-500 text-white">
            <Plus className="w-4 h-4 mr-2" />
            添加文件
          </Button>

          <Button onClick={handleOpenDirectory} size="lg" className="bg-blue-700 hover:bg-blue-600 text-white">
            <FolderPlus className="w-4 h-4 mr-2" />
            添加文件夹
          </Button>
        </div>
      </header>

      <Card className="flex-1 bg-card border-border shadow-md flex flex-col overflow-hidden min-h-0">
        <div className="flex-1 overflow-y-auto min-h-0">
          <Table>
            <TableHeader className="bg-muted/50 sticky top-0 z-10">
              <TableRow className="border-border hover:bg-muted/50">
                <TableHead className="text-muted-foreground w-[50px]">#</TableHead>
                <TableHead className="text-muted-foreground">文件名</TableHead>
                <TableHead className="text-muted-foreground w-[120px]">信息</TableHead>
                <TableHead className="text-muted-foreground w-[100px]">参数</TableHead>
                <TableHead className="text-muted-foreground w-[120px] text-center">状态</TableHead>
                <TableHead className="text-muted-foreground w-[180px] text-right">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {files.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={6} className="h-[400px] text-center text-muted-foreground">
                    <FileVideo className="w-16 h-16 mx-auto mb-4 opacity-20" />
                    <p>拖拽 MP4 文件到此处</p>
                    <p className="text-sm mt-2">或点击“添加文件”</p>
                    <p className="text-sm mt-2">或点击“添加文件”</p>
                  </TableCell>
                </TableRow>
              ) : (
                files.map((file, index) => (
                  <TableRow key={file.id + index} className="border-border hover:bg-muted/30 transition-colors">
                    <TableCell className="text-muted-foreground font-mono">{index + 1}</TableCell>
                    <TableCell className="font-medium text-foreground truncate max-w-[250px]" title={file.path}>
                      <div className="flex flex-col">
                        <span>{file.name}</span>
                        <span className="text-xs text-muted-foreground truncate">{file.path}</span>
                        {file.metadata?.modified && (
                          <span className="text-[10px] text-muted-foreground">
                            {new Date(file.metadata.modified).toLocaleDateString()} {new Date(file.metadata.modified).toLocaleTimeString()}
                          </span>
                        )}
                      </div>
                    </TableCell>
                    <TableCell className="text-muted-foreground text-sm">
                      {file.metadata ? (
                        <div className="flex flex-col gap-1">
                          <Badge variant="outline" className="w-fit text-xs border-border">{formatSize(file.metadata.size)}</Badge>
                          <span className="text-xs">{formatTime(file.metadata.duration)}</span>
                        </div>
                      ) : (
                        <span className="text-xs opacity-50">-</span>
                      )}
                    </TableCell>
                    <TableCell className="text-muted-foreground text-sm">
                      {file.metadata ? (
                        <div className="flex flex-col gap-1">
                          <span className="text-xs font-mono bg-muted px-1 rounded">{file.metadata.width}x{file.metadata.height}</span>
                          <span className="text-xs text-muted-foreground">{file.metadata.codec}</span>
                        </div>
                      ) : (
                        <span className="text-xs opacity-50">-</span>
                      )}
                    </TableCell>
                    <TableCell className="text-center">
                      <StatusBadge status={file.status} message={file.message} />
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex items-center justify-end gap-2">
                        <Button
                          size="sm"
                          variant="ghost"
                          className="h-8 w-8 p-0 text-blue-400 hover:text-blue-300 hover:bg-blue-900/20"
                          onClick={() => setPlayingFile(file)}
                          title="播放视频"
                        >
                          <Play className="w-4 h-4" />
                        </Button>
                        {file.status === 'unoptimized' && (
                          <Button
                            size="sm"
                            variant="secondary"
                            className="bg-emerald-500/10 text-emerald-400 hover:bg-emerald-500/20 border-emerald-500/20 border"
                            onClick={(e) => {
                              e.stopPropagation();
                              optimizeFile(file.path);
                            }}
                          >
                            <Zap className="w-3 h-3 mr-1" />
                            优化
                          </Button>
                        )}
                      </div>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>
      </Card>

      {/* Video Player Modal */}
      {
        playingFile && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-sm" onClick={() => setPlayingFile(null)}>
            <div className="bg-background border border-border rounded-lg shadow-2xl overflow-hidden max-w-5xl w-full max-h-[90vh] flex flex-col" onClick={e => e.stopPropagation()}>
              <div className="flex items-center justify-between p-4 border-b border-border">
                <h3 className="text-lg font-semibold truncate text-foreground pr-4">{playingFile.name}</h3>
                <Button variant="ghost" size="icon" onClick={() => setPlayingFile(null)} className="hover:bg-muted text-muted-foreground hover:text-foreground">
                  <X className="w-5 h-5" />
                </Button>
              </div>
              <div className="flex-1 bg-black flex items-center justify-center overflow-hidden relative">
                {/* 
                    Use /video/ prefix which Go backend handles.
                    We need strictly URL encoded path segments.
                    But Go's http.FileSystem usually expects straight paths?
                    Browser handles URL encoding.
                    If path is "C:\foo bar.mp4", URL should be "/video/C:/foo%20bar.mp4" (roughly).
                    Wait, windows paths in URL? 
                    Wails + AssetServer usually handles local paths if we map it right.
                    Let's try direct path first. 
                 */}
                <video
                  controls
                  autoPlay
                  className="w-full h-full object-contain"
                  src={`/video/${encodeURIComponent(playingFile.path)}`}
                >
                  您的浏览器不支持 HTML5 视频播放。
                </video>
              </div>
              <div className="p-4 bg-background text-sm text-muted-foreground border-t border-border flex justify-between">
                <span>{playingFile.metadata?.width}x{playingFile.metadata?.height}</span>
                <span>{playingFile.metadata?.codec}</span>
              </div>
            </div>
          </div>
        )
      }

      <footer className="mt-4 flex items-center justify-center text-muted-foreground text-xs border-t border-border/50 pt-4 relative">
        <div className="flex items-center gap-2">
          {/* Connection Status Dot */}
          <div className={`w-2 h-2 rounded-full ${wailsConnected ? 'bg-emerald-500' : 'bg-red-500 animate-pulse'}`} title={wailsConnected ? "Wails Connected" : "Wails Disconnected"} />

          {/* Fix double 'vv' if appVersion already contains 'v' */}
          {appVersion.startsWith('v') ? appVersion : `v${appVersion}`} • {theme === 'light' ? 'Light' : 'Dark'} Mode
        </div>
        {/* Hidden area for balancing if needed, or just centered absolutely */}
      </footer>
    </div >
  );
}

function StatusBadge({ status, message }: { status: FileStatus, message?: string }) {
  if (status === 'pending') return <Badge variant="outline" className="text-muted-foreground">等待中</Badge>;
  if (status === 'scanning') return <Badge variant="secondary" className="bg-blue-500/10 text-blue-500 dark:text-blue-400"><Loader2 className="w-3 h-3 mr-1 animate-spin" /> 检测中</Badge>;
  if (status === 'optimized') return <Badge variant="default" className="bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20 border"><CheckCircle2 className="w-3 h-3 mr-1" /> 已优化</Badge>;
  if (status === 'unoptimized') return <Badge variant="destructive" className="bg-red-500/10 text-red-600 dark:text-red-400 border-red-500/20 border"><XCircle className="w-3 h-3 mr-1" /> 未优化</Badge>;
  if (status === 'error') return <Badge variant="destructive" title={message}>错误</Badge>;
  return null;
}
