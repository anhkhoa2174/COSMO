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

const MIME_IMAGE_TYPES = ['image/gif', 'image/jpeg', 'image/png', 'image/svg+xml', 'image/webp'];

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
      setFilesUploaded((_filesUploaded) => [..._filesUploaded, ...acceptedFiles]);
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
          'flex justify-center items-center w-full h-32 border-dashed border-2 border-gray-200 rounded-lg hover:bg-accent hover:text-accent-foreground transition-all select-none cursor-pointer',
          dropZoneClassName
        )}
      >
        <input {...dropzone.getInputProps()} />
        {children ? (
          children(dropzone)
        ) : dropzone.isDragAccept ? (
          <div className="text-sm font-medium">Drop your files here!</div>
        ) : (
          <div className="flex items-center flex-col gap-1.5">
            <div className="flex items-center flex-row gap-0.5 text-sm font-medium">
              <Upload className="mr-2 h-4 w-4" /> Upload {props.multiple ? 'files' : 'file'}
            </div>
            {props.maxSize && (
              <div className="text-xs text-gray-400 font-medium">
                Max. file size: {(props.maxSize / (1024 * 1024)).toFixed(2)} MB
              </div>
            )}
          </div>
        )}
      </div>
      {showErrorMessage && errorMessage && (
        <span className="text-xs text-destructive mt-3">{errorMessage}</span>
      )}
      {showFilesList && filesUploaded.length > 0 && (
        <div
          className={`flex flex-col gap-2 w-full ${filesUploaded.length > 2 ? 'h-48' : 'h-fit'} mt-2 ${filesUploaded.length > 0 ? 'pb-2' : ''}`}
        >
          <div className="w-full">
            {filesUploaded.map((fileUploaded, index) => (
              <div
                key={index}
                className="flex justify-between items-center flex-row w-full h-16 mt-2 px-4 border-solid border-2 border-gray-200 rounded-lg shadow-sm"
              >
                <div className="flex items-center flex-row gap-4 h-full">
                  {MIME_IMAGE_TYPES.includes(fileUploaded.type) ? (
                    <Image className="text-rose-700 w-6 h-6" />
                  ) : (
                    <FileText className="text-rose-700 w-6 h-6" />
                  )}
                  <div className="flex flex-col gap-0">
                    <div className="text-[0.85rem] font-medium leading-snug truncate">
                      {fileUploaded.name.split('.').slice(0, -1).join('.')}
                    </div>
                    <div className="text-[0.7rem] text-gray-500 leading-tight">
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
