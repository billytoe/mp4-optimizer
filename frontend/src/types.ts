export type FileStatus = 'pending' | 'scanning' | 'optimized' | 'unoptimized' | 'error';

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
    metadata?: FileMetadata;
}
