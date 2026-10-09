// Packages:
import React, { useState } from 'react';
import { FileText, Image, Trash, Upload } from 'lucide-react';
// Typescript:
import {
  useDropzone,
  type DropzoneProps as _DropzoneProps,
  type DropzoneState as _DropzoneState,
} from 'react-dropzone';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';

export interface DropzoneState extends _DropzoneState {}

export interface DropzoneProps extends Omit<_DropzoneProps, 'children'> {
  files?: File[];
  containerClassName?: string;
  dropZoneClassName?: string;
  children?: (dropzone: DropzoneState) => React.ReactNode;
  showFilesList?: boolean;
  showErrorMessage?: boolean;
}

const MIME_IMAGE_TYPES = [
  'image/gif',
  'image/jpeg',
  'image/png',
  'image/svg+xml',
  'image/webp',
];

const FileDropzone = ({
  files = [],
  containerClassName,
  dropZoneClassName,
  children,
  showFilesList = true,
  showErrorMessage = true,
  ...props
}: DropzoneProps) => {
  // Constants:
  const dropzone = useDropzone({
    ...props,
    onDrop(acceptedFiles, fileRejections, event) {
      if (props.onDrop) {
        props.onDrop(acceptedFiles, fileRejections, event);
      }
      setFilesUploaded((_filesUploaded) => [
        ..._filesUploaded,
        ...acceptedFiles,
      ]);
      if (fileRejections.length > 0) {
        let _errorMessage = `Could not upload ${fileRejections[0].file.name}`;
        if (fileRejections.length > 1) {
          _errorMessage += `, and ${fileRejections.length - 1} other files.`;
        }
        setErrorMessage(_errorMessage);
      } else {
        setErrorMessage('');
      }
    },
  });

  // State:
  const [filesUploaded, setFilesUploaded] = useState<File[]>(files);
  const [errorMessage, setErrorMessage] = useState<string>();

  // Functions:
  const deleteUploadedFile = (index: number) => {
    const copyFilesUploaded = [...filesUploaded];
    const _uploadedFiles = [
      ...copyFilesUploaded.slice(0, index),
      ...copyFilesUploaded.slice(index + 1),
    ];
    if (props.onDrop) {
      props.onDrop(_uploadedFiles, [], {} as any);
    }
    setFilesUploaded(_uploadedFiles);
  };

  // Return:
  return (
    <div className={cn('flex flex-col gap-2', containerClassName)}>
      <div
        {...dropzone.getRootProps()}
        className={cn(
          'flex h-32 w-full cursor-pointer select-none items-center justify-center rounded-lg border-2 border-dashed border-gray-200 transition-all hover:bg-accent hover:text-accent-foreground',
          dropZoneClassName
        )}
      >
        <input {...dropzone.getInputProps()} />
        {children ? (
          children(dropzone)
        ) : dropzone.isDragAccept ? (
          <div className="text-sm font-medium">Drop your files here!</div>
        ) : (
          <div className="flex flex-col items-center gap-1.5">
            <div className="flex flex-row items-center gap-0.5 text-sm font-medium">
              <Upload className="mr-2 h-4 w-4" /> Upload{' '}
              {props.multiple ? 'files' : 'file'}
            </div>
            {props.maxSize && (
              <div className="text-xs font-medium text-gray-400">
                Max. file size: {(props.maxSize / (1024 * 1024)).toFixed(2)} MB
              </div>
            )}
          </div>
        )}
      </div>
      {showErrorMessage && errorMessage && (
        <span className="mt-3 text-xs text-destructive">{errorMessage}</span>
      )}
      {showFilesList && filesUploaded.length > 0 && (
        <div
          className={`flex w-full flex-col gap-2 ${filesUploaded.length > 2 ? 'h-48' : 'h-fit'} mt-2 ${filesUploaded.length > 0 ? 'pb-2' : ''}`}
        >
          <div className="w-full">
            {filesUploaded.map((fileUploaded, index) => (
              <div
                key={index}
                className="mt-2 flex h-16 w-full flex-row items-center justify-between rounded-lg border-2 border-solid border-gray-200 px-4 shadow-sm"
              >
                <div className="flex h-full flex-row items-center gap-4">
                  {MIME_IMAGE_TYPES.includes(fileUploaded.type) ? (
                    <Image className="h-6 w-6 text-rose-700" />
                  ) : (
                    <FileText className="h-6 w-6 text-rose-700" />
                  )}
                  <div className="flex flex-col gap-0">
                    <div className="truncate text-[0.85rem] font-medium leading-snug">
                      {fileUploaded.name.split('.').slice(0, -1).join('.')}
                    </div>
                    <div className="text-[0.7rem] leading-tight text-gray-500">
                      .{fileUploaded.name.split('.').pop()} •{' '}
                      {(fileUploaded.size / (1024 * 1024)).toFixed(2)} MB
                    </div>
                  </div>
                </div>
                <Button
                  variant="outline"
                  onClick={() => deleteUploadedFile(index)}
                  className="rounded-full"
                  size="icon"
                >
                  <Trash />
                </Button>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};

// Exports:
export default FileDropzone;
