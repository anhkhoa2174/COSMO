'use client';

import { toast } from 'sonner';
import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useMutation, useQuery } from '@tanstack/react-query';
import {
  ArrowRight,
  Check,
  CircleCheckBig,
  Info,
  SquareMinus,
} from 'lucide-react';

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { cn } from '@/lib/utils';
import { IconHubspot } from '@/assets/icons';
import { Label } from '@/components/ui/label';
import { HubspotApi } from '@/network/client/hubspot';
import { MainButton } from '@/components/buttons/main-button';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';

import type { HubspotProperty } from '@/network/client/hubspot';
import { ComboboxDemo } from './combobox';
import { getCustomFields } from '@/network/client/custom-field';

type MappingField = Record<string, string>;

const defaultFields = [
  { value: 'first_name', label: 'First Name', groupName: 'Default field' },
  { value: 'last_name', label: 'Last Name', groupName: 'Default field' },
  { value: 'email', label: 'Email', groupName: 'Default field' },
  { value: 'phone', label: 'Phone', groupName: 'Default field' },
  { value: 'company', label: 'Company', groupName: 'Default field' },
  { value: 'job_title', label: 'Job Title', groupName: 'Default field' },
  { value: 'address', label: 'Address', groupName: 'Default field' },
  { value: 'city', label: 'City', groupName: 'Default field' },
  { value: 'country', label: 'Country', groupName: 'Default field' },
  { value: 'state', label: 'State', groupName: 'Default field' },
  { value: 'zip', label: 'ZIP', groupName: 'Default field' },
];

function swapObjectKeys(obj: Record<string, string>) {
  return Object.fromEntries(
    Object.entries(obj).map(([key, value]) => [value, key])
  );
}

export default function HubspotIntegrationPage() {
  const {
    data: hubspotInfo,
    isSuccess,
    refetch,
  } = useQuery({
    queryKey: ['hubspot-me'],
    queryFn: () => HubspotApi.getMe(),
  });
  const accessToken = hubspotInfo?.data?.access_token;
  const refreshToken = hubspotInfo?.data?.refresh_token;
  const router = useRouter();
  const [active, setActive] = useState(0);
  const [activeCustomFields, setActiveCustomFields] = useState<
    HubspotProperty[]
  >([]);
  const [mappingField, setMappingField] = useState<MappingField>({});

  useEffect(() => {
    if (isSuccess) {
      setMappingField(swapObjectKeys(hubspotInfo.data?.field_mapping || {}));
    }
  }, [isSuccess]);

  // Step 1
  const ownersQuery = useQuery({
    queryFn: () => HubspotApi.getOwners(accessToken, refreshToken),
    queryKey: ['hubspotOwners'],
    enabled: !!accessToken && !!refreshToken,
  });

  const getHubspotAuth = async () => {
    try {
      const response = await HubspotApi.getAuthURL();
      const feature = `width=600,height=600,left=${(window.screen.width - 600) / 2},top=${(window.screen.height - 600) / 2}`;
      const authWindow = window.open(response.url, '_blank', feature);

      const handleMessage = (event: MessageEvent) => {
        if (event.data?.type === 'HUBSPOT_AUTH') {
          refetch();
          authWindow?.close();
          setActive(1);
          window.removeEventListener('message', handleMessage);
        }
      };

      window.addEventListener('message', handleMessage);
    } catch (error: any) {
      toast.error(error.message);
    }
  };

  // Step 2
  const customFieldsQuery = useQuery({
    queryKey: ['customFields'],
    queryFn: getCustomFields,
    enabled: active === 1,
  });

  const appFields = [
    ...defaultFields,
    ...(customFieldsQuery.data?.data.list.map((i) => ({
      value: i.normalized_name,
      label: i.name,
      groupName: 'Custom field',
    })) || []),
  ];

  const _appFields = appFields.reduce(
    (acc, item) => {
      const group = item.groupName;
      if (!acc[group]) {
        acc[group] = [];
      }
      acc[group].push(item);
      return acc;
    },
    {} as {
      [key: string]: { value: string; label: string; groupName: string }[];
    }
  );

  const propertiesQuery = useQuery({
    queryKey: ['hubspotProperties'],
    queryFn: () => HubspotApi.getProperties(accessToken, refreshToken),
    enabled: active === 1 && !!accessToken && !!refreshToken,
  });

  useEffect(() => {
    if (propertiesQuery.isSuccess) {
      const properties = propertiesQuery.data?.results;
      if (properties) {
        Object.entries(mappingField).forEach(([key, value]) => {
          if (!['email', 'firstname', 'lastname'].includes(key)) {
            const property = properties.find((p) => p.name === key);
            if (property) {
              setActiveCustomFields((prev) => [...prev, property]);
            }
          }
        });
      }
    }
  }, [propertiesQuery.isSuccess, propertiesQuery.data]);

  const standardFields =
    (propertiesQuery.data?.results.filter((p) =>
      ['email', 'firstname', 'lastname'].includes(p.name)
    ) as HubspotProperty[]) || [];
  const customFields =
    (propertiesQuery.data?.results.filter(
      (p) => !['email', 'firstname', 'lastname'].includes(p.name)
    ) as HubspotProperty[]) || [];

  const updateMutation = useMutation({
    mutationFn: (data: any) => HubspotApi.updateMe(data),
    onSuccess: () => {
      setActive((current) => current + 1);
    },
    onError: (error: any) => {
      toast.error(error.message);
    },
  });

  const saveMapping = async () => {
    const isValid =
      Object.values(mappingField).every((m) => m !== '') &&
      Object.entries(mappingField).length >= 3;
    if (!isValid) {
      toast.error('Please map all added fields');
      return;
    }
    await updateMutation.mutateAsync({
      field_mapping: swapObjectKeys(mappingField),
    });
  };

  const updateMapping = (hubspotField: string, appField: string) => {
    setMappingField((prev) => ({
      ...prev,
      [hubspotField]: appField,
    }));
  };

  const addCustomField = (name: string) => {
    if (activeCustomFields.some((field) => field.name === name)) {
      toast.error('Field already added');
      return;
    }
    const newField = customFields.find((field) => field.name === name);
    if (newField) {
      setActiveCustomFields((prev) => [...prev, newField]);
      setMappingField((prev) => ({
        ...prev,
        [newField.name]: '',
      }));
    }
  };

  const removeCustomField = (name: string) => {
    setActiveCustomFields((prev) =>
      prev.filter((field) => field.name !== name)
    );
    setMappingField((prev) => {
      const newMapping = { ...prev };
      delete newMapping[name];
      return newMapping;
    });
  };

  return (
    <div className="container mx-auto space-y-4 py-8">
      <div className="mx-auto max-w-lg text-2xl font-bold">
        HubSpot Integration
      </div>
      <div className="flex items-center justify-center gap-2">
        {[0, 1, 2].map((step) => (
          <div key={step} className="flex items-center">
            <div
              className={cn(
                'flex h-10 w-10 items-center justify-center rounded-full border-2',
                active === step
                  ? 'border-blue-400 bg-blue-400 text-primary-foreground'
                  : active > step
                    ? 'border-blue-500 bg-blue-500 text-primary-foreground'
                    : 'border-border'
              )}
            >
              {active > step ? <Check className="h-5 w-5" /> : step + 1}
            </div>
            {step < 2 && (
              <div
                className={cn(
                  'ml-2 h-1 w-32',
                  active > step ? 'bg-blue-500' : 'bg-border'
                )}
              />
            )}
          </div>
        ))}
      </div>
      <div className="flex items-center justify-center gap-[130px]">
        {['Connect', 'Map Fields', 'Complete'].map((step, index) => (
          <div key={index} className="flex items-center">
            {step}
          </div>
        ))}
      </div>
      <div className="mx-auto max-w-lg space-y-8">
        {/* Step 1 */}
        {active === 0 && (
          <Card>
            <CardContent className="space-y-4 pt-4">
              <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-lg bg-orange-50">
                <IconHubspot />
              </div>
              <div className="text-center">
                <p className="text-base font-medium">
                  Connect your HubSpot account
                </p>
                <p className="text-muted-foreground">
                  Authorize access to import contacts from your HubSpot account
                </p>
              </div>
              {accessToken ? (
                <Alert>
                  <CircleCheckBig className="h-4 w-4" />
                  <AlertTitle>Connected to HubSpot</AlertTitle>
                  <AlertDescription>
                    Account: {ownersQuery.data?.results[0]?.email || 'N/A'}
                  </AlertDescription>
                </Alert>
              ) : (
                <Alert>
                  <Info className="h-4 w-4" />
                  <AlertTitle>Required Permissions</AlertTitle>
                  <AlertDescription>
                    This integration needs access to contacts and lists in your
                    HubSpot account.
                  </AlertDescription>
                </Alert>
              )}
              <div className="flex flex-col">
                <MainButton
                  text={
                    accessToken
                      ? 'Connect another account'
                      : 'Connect to HubSpot'
                  }
                  onClick={getHubspotAuth}
                />
              </div>
              <div className="text-center text-xs text-muted-foreground">
                You'll be redirected to HubSpot to authorize this connection
              </div>
              {accessToken && (
                <div className="flex flex-col">
                  <button
                    className="mx-auto underline underline-offset-2"
                    onClick={() => setActive(1)}
                  >
                    Continue mapping
                  </button>
                </div>
              )}
            </CardContent>
          </Card>
        )}

        {/* Step 2 */}
        {active === 1 && (
          <Card>
            <CardHeader>
              <CardTitle>Map Contact Fields</CardTitle>
              <CardDescription>
                Configure how HubSpot fields map to your platform fields
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <Alert>
                <Info className="h-4 w-4" />
                <AlertTitle>Default Mappings</AlertTitle>
                <AlertDescription>
                  We've set up default mappings for standard fields. You can
                  customize these and add additional fields.
                </AlertDescription>
              </Alert>

              {propertiesQuery.isLoading ? (
                <div className="italic">HubSpot properties loading...</div>
              ) : (
                <>
                  <div className="space-y-4">
                    <h3 className="text-base font-medium">Standard Fields</h3>
                    <div className="space-y-4">
                      {standardFields.map((field) => (
                        <div
                          key={field.name}
                          className="grid grid-cols-12 items-center gap-4 rounded-sm border bg-zinc-50 px-4 py-3"
                        >
                          <div className="col-span-6">
                            <Label>
                              {field.label}{' '}
                              <span className="text-destructive">*</span>
                            </Label>
                            <p className="line-clamp-1 text-muted-foreground">
                              {field.description}
                            </p>
                          </div>
                          <div className="col-span-1">
                            <ArrowRight className="h-4 w-4 text-muted-foreground" />
                          </div>
                          <div className="col-span-5">
                            <Select
                              value={mappingField[field.name]}
                              onValueChange={(value) =>
                                updateMapping(field.name, value)
                              }
                            >
                              <SelectTrigger>
                                <SelectValue placeholder="Select Cosmo field" />
                              </SelectTrigger>
                              <SelectContent>
                                {Object.entries(_appFields).map(
                                  ([group, items]) => (
                                    <SelectGroup key={group}>
                                      <SelectLabel>{group}</SelectLabel>
                                      {items.map((p) => (
                                        <SelectItem
                                          key={p.value}
                                          value={p.value}
                                        >
                                          {p.label}
                                        </SelectItem>
                                      ))}
                                    </SelectGroup>
                                  )
                                )}
                              </SelectContent>
                            </Select>
                          </div>
                        </div>
                      ))}
                      {standardFields.length === 0 && (
                        <p className="text-center italic text-muted-foreground">
                          No Standard Fields Found
                        </p>
                      )}
                    </div>
                  </div>
                  <div className="space-y-4">
                    <h3 className="text-base font-medium">Custom Fields</h3>
                    <div className="space-y-4">
                      {activeCustomFields.map((field) => (
                        <div
                          key={field.name}
                          className="grid grid-cols-12 items-center gap-4 rounded-sm border border-dashed px-4 py-3"
                        >
                          <div className="col-span-6 flex items-center gap-4">
                            <button
                              onClick={() => removeCustomField(field.name)}
                            >
                              <SquareMinus className="h-4 w-4" />
                            </button>
                            <div>
                              <Label>{field.label}</Label>
                              <p
                                className="line-clamp-1 text-muted-foreground"
                                title={field.description}
                              >
                                {field.description}
                              </p>
                            </div>
                          </div>
                          <div className="col-span-1">
                            <ArrowRight className="h-4 w-4 text-muted-foreground" />
                          </div>
                          <div className="col-span-5">
                            <Select
                              value={mappingField[field.name]}
                              onValueChange={(value) =>
                                updateMapping(field.name, value)
                              }
                            >
                              <SelectTrigger>
                                <SelectValue placeholder="Select Cosmo field" />
                              </SelectTrigger>
                              <SelectContent>
                                {Object.entries(_appFields).map(
                                  ([group, items]) => (
                                    <SelectGroup key={group}>
                                      <SelectLabel>{group}</SelectLabel>
                                      {items.map((p) => (
                                        <SelectItem
                                          key={p.value}
                                          value={p.value}
                                        >
                                          {p.label}
                                        </SelectItem>
                                      ))}
                                    </SelectGroup>
                                  )
                                )}
                              </SelectContent>
                            </Select>
                          </div>
                        </div>
                      ))}
                      {customFields.length === 0 && (
                        <p className="text-center italic text-muted-foreground">
                          No Custom Fields Found
                        </p>
                      )}
                      <ComboboxDemo
                        data={customFields}
                        onSelect={addCustomField}
                      />
                    </div>
                  </div>
                </>
              )}

              <div className="flex w-full items-center justify-between gap-2">
                <MainButton
                  variant="outline"
                  onClick={() => setActive((prev) => prev - 1)}
                  className="w-full"
                  text="Back"
                />
                <MainButton
                  onClick={saveMapping}
                  className="w-full"
                  text="Save"
                  loading={updateMutation.isPending}
                />
              </div>
            </CardContent>
          </Card>
        )}

        {/* Step 3 */}
        {active === 2 && (
          <Card>
            <CardContent className="space-y-8 pt-4">
              <div className="flex flex-col space-y-1.5 text-center">
                <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-green-100">
                  <CircleCheckBig className="h-6 w-6 text-green-500" />
                </div>
                <h3 className="text-base font-semibold leading-none tracking-tight">
                  Setup Complete!
                </h3>
                <p className="text-sm text-muted-foreground">
                  Your HubSpot account is now connected and ready to use
                </p>
              </div>
              <div className="">
                <div className="mb-4 rounded-lg border border-green-100 bg-green-50 p-4">
                  <div className="flex items-start">
                    <CircleCheckBig className="mr-2 mt-0.5 h-5 w-5 flex-shrink-0 text-green-500" />
                    <div>
                      <h3 className="font-medium">Connected to HubSpot</h3>
                      <p className="text-sm text-gray-600">
                        Account: {ownersQuery.data?.results[0]?.email || 'N/A'}
                      </p>
                    </div>
                  </div>
                </div>
                <div className="space-y-2 text-sm">
                  <p className="font-medium">What you can do now:</p>
                  <ul className="list-disc space-y-1 pl-5">
                    <li>Import contact lists from HubSpot</li>
                    <li>Use HubSpot contacts in your campaigns</li>
                    <li>Keep your contacts in sync</li>
                  </ul>
                </div>
              </div>
              <div className="flex items-center">
                <MainButton
                  text="Go to Contact Lists"
                  onClick={() => router.push('/audiences')}
                  className="w-full"
                />
              </div>
            </CardContent>
          </Card>
        )}
      </div>
    </div>
  );
}
