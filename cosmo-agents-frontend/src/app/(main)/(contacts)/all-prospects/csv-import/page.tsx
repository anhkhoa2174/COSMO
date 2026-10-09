'use client';

import { useEffect, useState } from 'react';
import { useMutation, useQuery } from '@tanstack/react-query';
import { ArrowLeft, ArrowRight } from 'lucide-react';
import { toast } from 'sonner';
import { MainButton } from '@/components/buttons/main-button';
import FileDropzone from '@/components/file-dropzone';
import { Badge } from '@/components/ui/badge';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { cn } from '@/lib/utils';
import {
  SPREADSHEET_EXTENSIONS,
  toCsvFile,
} from '@/lib/import/spreadsheet';
import {
  extractCsvHeaders,
  ExtractHeadersResponse,
  importCSV,
} from '@/network/client/contact';
import { getCustomFields } from '@/network/client/custom-field';
import { useOperationFlow } from '@/network/client/template';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { ContentLayout } from '@/components/nav/content-layout';
import type { StepItem } from '@/components/nyxb-ui/stepper';
import { Step, Stepper, useStepper } from '@/components/nyxb-ui/stepper';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { useRouter } from 'next/navigation';
import { Progress } from '@/components/ui/progress';
import { BackButton } from '@/components/buttons/back-button';

const DEFAULT_FIELDS = [
  { field: 'first_name', label: 'First Name', is_required: true },
  { field: 'last_name', label: 'Last Name', is_required: true },
  { field: 'email', label: 'Email', is_required: true },
  { field: 'phone', label: 'Phone', is_required: false },
  { field: 'company', label: 'Company', is_required: false },
  { field: 'job_title', label: 'Job Title', is_required: false },
  { field: 'industry', label: 'Industry', is_required: false },
  { field: 'contact_channel', label: 'Contact Channel', is_required: false },
  { field: 'address', label: 'Address', is_required: false },
  { field: 'city', label: 'City', is_required: false },
  { field: 'country', label: 'Country', is_required: false },
  { field: 'state', label: 'State', is_required: false },
  { field: 'zip', label: 'ZIP', is_required: false },
];

const steps = [
  { label: 'Upload CSV' },
  { label: 'Field Mapping' },
  { label: 'Confirm' },
  { label: 'Complete' },
] satisfies StepItem[];

type Mapping = Record<string, string>;

export default function CSVImportPage() {
  const router = useRouter();

  const [files, setFiles] = useState<File[]>([]);
  const [mapping, setMapping] = useState<Mapping>({});
  const [nameImport, setNameImport] = useState('');
  const [headers, setHeaders] = useState<string[]>([]);

  const { data } = useQuery({
    queryKey: ['customFields'],
    queryFn: getCustomFields,
  });

  const fields = [
    ...DEFAULT_FIELDS,
    ...(data?.data.list.map((field) => ({
      field: field.normalized_name,
      label: field.name,
      data_type: field.data_type,
      is_required: field.is_required,
    })) ?? []),
  ];

  const handleUploadSuccess = (data: ExtractHeadersResponse) => {
    const _mapping = data.system.reduce(
      (acc, field) => ({ ...acc, [field.mapping]: field.name }),
      {}
    );
    setMapping(_mapping);
    setHeaders([
      ...data.custom.flatMap((field) => field.name),
      ...data.system.map((field) => field.name),
    ]);
    setNameImport(files[0]?.name ?? '');
  };

  const handleMappingChange = (field: string, value: string) => {
    setMapping((prev) => ({ ...prev, [field]: value }));
  };

  return (
    <ContentLayout
      title="CSV Import"
      className="container mx-auto"
      leftSection={<BackButton href="/all-prospects" />}
    >
      <Stepper
        initialStep={0}
        steps={steps}
        styles={{
          'step-button-container': cn(
            'border-none bg-zinc-300 text-zinc-500',
            'data-[current=true]:bg-blue-500 data-[current=true]:text-white',
            'data-[active=true]:bg-blue-400 data-[active=true]:text-white'
          ),
          'horizontal-step':
            'data-[completed=true]:[&:not(:last-child)]:after:bg-blue-400',
        }}
      >
        <Step label="Upload CSV">
          <UploadCSVStep
            files={files}
            onFilesChange={setFiles}
            onSuccess={handleUploadSuccess}
          />
        </Step>
        <Step label="Field Mapping">
          <FieldMappingStep
            mapping={mapping}
            headers={headers}
            fields={fields}
            onMappingChange={handleMappingChange}
          />
        </Step>
        <Step label="Confirm">
          <ConfirmStep
            mapping={mapping}
            fields={fields}
            nameImport={nameImport}
            onNameImportChange={setNameImport}
          />
        </Step>
        <Step label="Complete">
          <CompleteStep
            mapping={mapping}
            files={files}
            nameImport={nameImport}
          />
        </Step>
      </Stepper>
    </ContentLayout>
  );
}

function UploadCSVStep({
  files,
  onFilesChange,
  onSuccess,
}: {
  files: File[];
  onFilesChange: (files: File[]) => void;
  onSuccess: (data: ExtractHeadersResponse) => void;
}) {
  const { nextStep } = useStepper();

  const extractHeadersMutation = useMutation({
    // Spreadsheets become CSV here rather than on the server, so the rest
    // of this flow keeps dealing with the one format it already knows.
    mutationFn: async (file: File) => extractCsvHeaders(await toCsvFile(file)),
    onSuccess: ({ data }) => {
      onSuccess(data);
      nextStep();
    },
    onError: (error) => toast.error(error.message),
  });

  return (
    <div className="space-y-8">
      <FileDropzone
        files={files}
        onDrop={onFilesChange}
        maxSize={5 * 1024 * 1024}
        accept={{
          'text/csv': ['.csv'],
          'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet': [
            '.xlsx',
          ],
          'application/vnd.ms-excel': ['.xls'],
        }}
        maxFiles={1}
        multiple={false}
      />
      <div className="flex justify-end">
        <MainButton
          text="Next"
          rightIcon={ArrowRight}
          disabled={files.length === 0}
          loading={extractHeadersMutation.isPending}
          onClick={() => files[0] && extractHeadersMutation.mutate(files[0])}
        />
      </div>
    </div>
  );
}

function FieldMappingStep({
  mapping,
  headers,
  fields,
  onMappingChange,
}: {
  mapping: Mapping;
  headers: string[];
  fields: any[];
  onMappingChange: (field: string, value: string) => void;
}) {
  const [requiredMessage, setRequiredMessage] = useState<{
    message: string;
  } | null>(null);
  const { prevStep, nextStep } = useStepper();

  const customFields = fields.filter((field) => field.data_type);

  const requiredFields = [
    ...DEFAULT_FIELDS.filter((field) => field.is_required).map(
      (field) => field.field
    ),
    ...customFields
      .filter((field) => field.is_required)
      .map((field) => field.field),
  ];

  const handleValidate = () => {
    if (requiredFields.every((field) => mapping[field])) {
      setRequiredMessage(null);
      nextStep();
    } else {
      setRequiredMessage({
        message: 'Please select a column for the required fields',
      });
    }
  };

  return (
    <div className="space-y-8">
      <Tabs defaultValue="system">
        <TabsList className="mb-4 w-full">
          <TabsTrigger value="system" className="flex-1">
            System Fields
          </TabsTrigger>
          <TabsTrigger value="custom" className="flex-1">
            Custom Fields
          </TabsTrigger>
        </TabsList>
        {requiredMessage && (
          <p className="text-destructive">{requiredMessage.message}</p>
        )}
        <TabsContent value="system" className="space-y-4">
          {DEFAULT_FIELDS.map((field) => (
            <div key={field.field} className="flex items-center gap-4">
              <span className="w-32">
                {field.label}{' '}
                {requiredFields.includes(field.field) && (
                  <span className="text-destructive">*</span>
                )}
              </span>
              <Select
                value={mapping[field.field]}
                onValueChange={(value) => onMappingChange(field.field, value)}
              >
                <SelectTrigger className="w-full">
                  <SelectValue placeholder="Select a CSV column" />
                </SelectTrigger>
                <SelectContent>
                  {headers.map((header) => (
                    <SelectItem key={header} value={header}>
                      {header}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          ))}
        </TabsContent>
        <TabsContent value="custom" className="space-y-4">
          {customFields.map((field) => (
            <div key={field.id} className="flex items-center gap-4">
              <span className="w-32">
                {field.label}{' '}
                <span className="text-xs text-muted-foreground">
                  ({field.data_type})
                </span>
                {requiredFields.includes(field.field) && (
                  <span className="text-destructive">*</span>
                )}
              </span>
              <Select
                value={mapping[field.field]}
                onValueChange={(value) => onMappingChange(field.field, value)}
              >
                <SelectTrigger className="w-full">
                  <SelectValue placeholder="Select a CSV column" />
                </SelectTrigger>
                <SelectContent>
                  {headers.map((header) => (
                    <SelectItem key={header} value={header}>
                      {header}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          ))}
          {customFields.length === 0 && (
            <p className="text-center text-muted-foreground">
              No custom fields found
            </p>
          )}
        </TabsContent>
      </Tabs>
      <div className="flex justify-between gap-2">
        <MainButton
          onClick={prevStep}
          variant="secondary"
          text="Back"
          icon={ArrowLeft}
        />
        <MainButton
          onClick={handleValidate}
          disabled={Object.keys(mapping).length === 0}
          text="Next"
          rightIcon={ArrowRight}
        />
      </div>
    </div>
  );
}

function ConfirmStep({
  mapping,
  fields,
  nameImport,
  onNameImportChange,
}: {
  mapping: Mapping;
  fields: any[];
  nameImport: string;
  onNameImportChange: (name: string) => void;
}) {
  const { prevStep, nextStep } = useStepper();

  const getMappedFields = () => [
    ...fields.filter((field) => mapping[field.field]),
  ];
  const getUnmappedFields = () => [
    ...fields.filter((field) => !mapping[field.field]),
  ];

  return (
    <div className="space-y-8">
      <div className="space-y-2">
        <Label>Name Import</Label>
        <Input
          value={nameImport}
          onChange={(e) => onNameImportChange(e.target.value)}
          placeholder="Name of the import"
        />
        <p className="text-[0.8rem] text-muted-foreground">
          After import, new contact list will be created with this name.
        </p>
      </div>
      <div className="space-y-2">
        <Label>Mapped Fields</Label>
        <div className="grid grid-cols-2 gap-4">
          {getMappedFields().map((field) => (
            <div key={field.field} className="flex items-center gap-2">
              <Badge variant="secondary" className="p-2">
                {field.label} {field.data_type && '*'}
              </Badge>
              <ArrowRight className="h-4 w-4 text-muted-foreground" />
              <Badge variant="secondary" className="p-2">
                {mapping[field.field]}
              </Badge>
            </div>
          ))}
        </div>
      </div>
      <div className="space-y-2">
        <Label>Unmapped Fields</Label>
        <div className="flex items-center gap-2">
          {getUnmappedFields().map((field) => (
            <Badge key={field.field} variant="secondary" className="p-2">
              {field.label} {field.data_type && '*'}
            </Badge>
          ))}
        </div>
      </div>
      <p className="text-sm text-muted-foreground">
        (*) indicates that the field is custom field
      </p>
      <div className="flex justify-between gap-2">
        <MainButton
          onClick={prevStep}
          variant="secondary"
          text="Back"
          icon={ArrowLeft}
        />
        <MainButton onClick={nextStep} text="Import" rightIcon={ArrowRight} />
      </div>
    </div>
  );
}

function CompleteStep({
  mapping,
  files,
  nameImport,
}: {
  mapping: Mapping;
  files: File[];
  nameImport: string;
}) {
  const { prevStep } = useStepper();
  const router = useRouter();

  const { mutate, isLoading, error, data } = useOperationFlow((request: File) =>
    toCsvFile(request).then((csv) =>
      importCSV(csv, { ...mapping, name_import: nameImport })
    )
  );

  const [progress, setProgress] = useState(0);

  useEffect(() => {
    files[0] && mutate(files[0]);
  }, []);

  useEffect(() => {
    if (error || progress >= 100) return;
    if (isLoading && progress >= 99) return;
    if (!isLoading && progress >= 4 && progress < 100) setProgress(99);

    const interval = setInterval(() => {
      setProgress(progress + 1);
    }, 250);
    return () => clearInterval(interval);
  }, [progress, error, isLoading]);

  const handleTryAgain = () => {
    setProgress(0);
    files[0] && mutate(files[0]);
  };

  return (
    <div className="space-y-8">
      <div className="space-y-4">
        <p className="font-semibold text-muted-foreground">
          Your import is in progress. It might take a few minutes to complete -{' '}
          <span
            className={cn(
              error
                ? 'text-destructive'
                : data?.data.status === 'success'
                  ? 'text-green-500'
                  : 'text-blue-500'
            )}
          >
            {progress}%
          </span>
        </p>
        <Progress
          value={progress}
          className={cn(
            error
              ? '[&>*:first-child]:data-[state=indeterminate]:!bg-destructive'
              : data?.data.status === 'success'
                ? '[&>*:first-child]:data-[state=indeterminate]:!bg-green-500'
                : '[&>*:first-child]:data-[state=indeterminate]:!bg-blue-500'
          )}
        />
        {error && (
          <p className="font-medium text-destructive">
            An error occurred while importing contacts: "
            <span className="uppercase">
              {(error as any).error?.message || error.message}
            </span>
            "
          </p>
        )}
        {data?.data.status === 'success' && (
          <p className="font-medium text-green-500">
            Contacts imported successfully
          </p>
        )}
      </div>
      {error && (
        <div className="flex justify-between gap-2">
          <MainButton
            onClick={prevStep}
            text="Back"
            variant="secondary"
            icon={ArrowLeft}
          />
          <MainButton text="Try again" onClick={handleTryAgain} />
        </div>
      )}
      {data?.data.status === 'success' && (
        <MainButton
          onClick={() => router.push('/all-prospects')}
          text="Back to contacts"
          className="float-right"
        />
      )}
    </div>
  );
}
