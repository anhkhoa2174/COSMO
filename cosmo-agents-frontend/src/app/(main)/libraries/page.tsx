'use client';

import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { toast } from 'sonner';
import { MainButton } from '@/components/buttons/main-button';
import { ArrowUpFromLine, Eye, Search } from 'lucide-react';
import { Separator } from '@/components/ui/separator';
import { DeleteButton } from '@/components/buttons/delete-button';
import { ContentLayout } from '@/components/nav/content-layout';
import { PageHero } from '@/components/ui/page-hero';
import { Library } from 'lucide-react';
import { KnowledgeApiV2 } from '@/network/client/knowledge';
import PDFPreviewModal from './_components/PDFPreviewModal';
import UploadPDFModal from './_components/UploadPDFModal';
import { Skeleton } from '@/components/ui/skeleton';

export default function LibrariesPage() {
  const [uploadModalOpened, setUploadModalOpened] = useState(false);
  const [loadingDelete, setLoadingDelete] = useState(false);
  const [selectedFile, setSelectedFile] = useState<{
    fileName: string;
    pdfBlob: Blob;
  } | null>(null);

  const openUploadModal = () => {
    setUploadModalOpened(true);
  };

  const closeUploadModal = () => {
    setUploadModalOpened(false);
  };

  const closePreviewModal = () => {
    setSelectedFile(null);
  };

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['knowledge'],
    queryFn: () => KnowledgeApiV2.list({ limit: 10, offset: 0 }),
  });

  const handleConfirm = async (files: File[]) => {
    if (files.length === 0) {
      return;
    }

    closeUploadModal();

    // The modal has already uploaded the files, with the type the user chose.
    // Uploading them again here sent every file twice and reset its type.
    localStorage.setItem('upload', 'success');
    toast.success('Import PDF successfully');
    refetch();
  };

  const handleConfirmDelete = async (id: string | null) => {
    if (!id) {
      return;
    }

    setLoadingDelete(true);

    try {
      await KnowledgeApiV2.delete(id);
      localStorage.setItem('delete', 'success');
      refetch();
    } catch (err: any) {
      toast.error(
        err.error?.message || err.message || 'Oops! Something went wrong'
      );
    } finally {
      setLoadingDelete(false);
    }
  };

  const handlePreview = async (file_name: string, s3_key: string) => {
    if (s3_key) {
      setLoadingDelete(true);

      try {
        const key = encodeURIComponent(s3_key);
        const blob = await KnowledgeApiV2.getFile(key);
        setSelectedFile({ fileName: file_name, pdfBlob: blob });
      } catch (err: any) {
        toast.error(
          err.error?.message || err.message || 'Oops! Something went wrong'
        );
      } finally {
        setLoadingDelete(false);
      }
    }
  };

  const rightSection = (
    <MainButton
      variant="outline"
      text="Upload Content"
      icon={ArrowUpFromLine}
      onClick={openUploadModal}
    />
  );

  return (
    <ContentLayout title="Knowledge Base" section="Knowledge" icon={Library}>
      <PageHero
        icon={Library}
        eyebrow="Knowledge"
        accent="emerald"
        title="Knowledge Base"
        description="Documents the AI searches when drafting. Each upload is chunked, embedded and indexed for retrieval."
        actions={rightSection}
      />
      <div className="flex flex-col items-center justify-center gap-4">
        {isLoading && (
          <div className="flex w-full items-center gap-4">
            <Skeleton className="h-[96px] w-[96px] rounded-lg" />
            <div className="flex-1 space-y-2">
              <Skeleton className="h-6" />
              <Skeleton className="h-16" />
            </div>
          </div>
        )}
        {data && data.data.length < 1 && (
          <div className="flex items-center justify-center">
            <Search />
            <p className="text-muted-foreground">No items to display</p>
          </div>
        )}
        {data?.data.map((knowledge) => (
          <div key={knowledge.id} className="space-y-4">
            <div className="flex items-center justify-center gap-4">
              <div
                className="group flex h-[128px] w-[128px] flex-shrink-0 cursor-pointer items-center justify-center rounded-lg bg-zinc-100"
                onClick={() =>
                  handlePreview(
                    knowledge?.cmetadata?.origin?.filename,
                    knowledge?.cmetadata?.s3_key
                  )
                }
              >
                <span className="text-2xl font-bold text-destructive group-hover:hidden">
                  PDF
                </span>
                <Eye className="absolute hidden text-destructive group-hover:block" />
              </div>
              <div className="space-y-2">
                <div className="text-lg font-semibold">
                  {knowledge?.cmetadata?.origin?.filename}
                </div>
                <div className="line-clamp-3 text-muted-foreground">
                  {knowledge?.summary_pair}
                </div>
                <div className="flex items-center gap-4">
                  <span className="text-blue-500">
                    {new Date(knowledge?.updated_at).toLocaleDateString()}
                  </span>
                  <span className="text-muted-foreground">
                    {(
                      (knowledge?.cmetadata?.origin?.file_size ?? 1) /
                      (1024 * 1024)
                    ).toFixed(2)}{' '}
                    MB
                  </span>
                </div>
              </div>
              <div className="flex items-center gap-2">
                <DeleteButton
                  size="icon"
                  className="flex-shrink-0"
                  variant="destructive-secondary"
                  onConfirm={() => handleConfirmDelete(knowledge?.id)}
                  description="This action cannot be undone. This will permanently delete your knowledge and remove
your data from our servers."
                  loading={loadingDelete}
                />
              </div>
            </div>
            <Separator />
          </div>
        ))}
      </div>
      <UploadPDFModal
        opened={uploadModalOpened}
        onClose={closeUploadModal}
        handleConfirm={handleConfirm}
      />
      {selectedFile && (
        <PDFPreviewModal
          opened={!!selectedFile}
          onClose={closePreviewModal}
          file={selectedFile}
        />
      )}
    </ContentLayout>
  );
}
