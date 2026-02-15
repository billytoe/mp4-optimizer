export type FileStatus = 'pending' | 'scanning' | 'optimizing' | 'optimized' | 'unoptimized' | 'error';

export interface FileMetadata {
    size: number;
    duration: number;
    width: number;
    height: number;
    codec: string;
    modified: string; // ISO string from Go time.Time
}

export interface FileItem {
    id: string; // unique id (path usually)
    path: string;
    name: string;
    size: number;
    status: FileStatus;
    message?: string;
    progress?: number;
    progressMessage?: string;
    metadata?: FileMetadata;
    isTruncated?: boolean; // 文件是否被截断/不完整
}

export interface ProgressEvent {
    path: string;
    progress: number;
    message: string;
}
