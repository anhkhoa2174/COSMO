'use client';

import { useRef, useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { format } from 'date-fns';
import { toast } from 'sonner';
import { ArrowUpFromLine, Download, FileIcon, Trash2 } from 'lucide-react';

import { ContentLayout } from '@/components/nav/content-layout';
import { PageHero } from '@/components/ui/page-hero';
import { FolderOpen } from 'lucide-react';
import { Card, CardContent } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { MainButton } from '@/components/buttons/main-button';
import FileApi from '@/network/client/file';
import type { FileItem } from '@/network/client/file';

function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  return `${(bytes / Math.pow(1024, i)).toFixed(i === 0 ? 0 : 1)} ${units[i]}`;
}

export default function FilesPage() {
  const queryClient = useQueryClient();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [downloadingKey, setDownloadingKey] = useState<string | null>(null);
  const [deletingKey, setDeletingKey] = useState<string | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ['files'],
    queryFn: () => FileApi.list(),
  });

  const uploadMutation = useMutation({
    mutationFn: (file: File) => FileApi.upload(file),
    onSuccess: () => {
      toast.success('File uploaded successfully');
      queryClient.invalidateQueries({ queryKey: ['files'] });
    },
    onError: () => {
      toast.error('Failed to upload file');
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (key: string) => FileApi.delete(key),
    onSuccess: () => {
      toast.success('File deleted successfully');
      queryClient.invalidateQueries({ queryKey: ['files'] });
    },
    onError: () => {
      toast.error('Failed to delete file');
    },
    onSettled: () => {
      setDeletingKey(null);
    },
  });

  const handleDelete = (key: string) => {
    if (!confirm('Are you sure you want to delete this file?')) return;
    setDeletingKey(key);
    deleteMutation.mutate(key);
  };

  const handleUpload = () => {
    fileInputRef.current?.click();
  };

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      uploadMutation.mutate(file);
      e.target.value = '';
    }
  };

  const handleDownload = async (key: string, fileName: string) => {
    setDownloadingKey(key);
    try {
      const res = await FileApi.getPresignedUrl(key);
      const url = res.data.url;
      const a = document.createElement('a');
      a.href = url;
      a.download = fileName;
      a.target = '_blank';
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
    } catch {
      toast.error('Failed to get download link');
    } finally {
      setDownloadingKey(null);
    }
  };

  const files = data?.data || [];

  const rightSection = (
    <MainButton
      variant="outline"
      text="Upload File"
      icon={ArrowUpFromLine}
      onClick={handleUpload}
      loading={uploadMutation.isPending}
    />
  );

  return (
    <ContentLayout title="Files" section="Knowledge" icon={FolderOpen}>
      <PageHero
        icon={FolderOpen}
        eyebrow="Knowledge"
        accent="blue"
        title="Files"
        description="Attachments and uploads stored against your organization."
        actions={rightSection}
      />
      <input
        ref={fileInputRef}
        type="file"
        className="hidden"
        onChange={handleFileChange}
      />

      {isLoading && (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {[1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-[120px] rounded-lg" />
          ))}
        </div>
      )}

      {!isLoading && files.length === 0 && (
        <div className="flex items-center justify-center py-12 text-muted-foreground">
          No files uploaded yet.
        </div>
      )}

      {!isLoading && files.length > 0 && (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {files.map((file: FileItem) => (
            <Card key={file.key}>
              <CardContent className="flex items-center gap-4 p-4">
                <div className="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-lg bg-muted">
                  <FileIcon className="h-5 w-5 text-muted-foreground" />
                </div>
                <div className="min-w-0 flex-1">
                  <p className="truncate text-sm font-medium">
                    {file.file_name}
                  </p>
                  <div className="flex items-center gap-2 text-xs text-muted-foreground">
                    <span>{formatFileSize(file.size)}</span>
                    {file.content_type && (
                      <>
                        <span>·</span>
                        <span>{file.content_type}</span>
                      </>
                    )}
                    {file.last_modified && (
                      <>
                        <span>·</span>
                        <span>
                          {format(new Date(file.last_modified), 'MMM d, yyyy')}
                        </span>
                      </>
                    )}
                  </div>
                </div>
                <MainButton
                  size="icon"
                  variant="ghost"
                  icon={Download}
                  onClick={() => handleDownload(file.key, file.file_name)}
                  loading={downloadingKey === file.key}
                />
                <MainButton
                  size="icon"
                  variant="ghost"
                  icon={Trash2}
                  onClick={() => handleDelete(file.key)}
                  loading={deletingKey === file.key}
                  className="text-destructive hover:text-destructive"
                />
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </ContentLayout>
  );
}
