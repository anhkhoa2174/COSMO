'use client';
import { Card, CardContent } from '@/components/ui/card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Textarea } from '@/components/ui/textarea';
import { generateId } from '@/helpers';
import useIsScroll from '@/hooks/use-is-scroll';
import { cn } from '@/lib/utils';
import { deleteCustomField, getCustomFields } from '@/network/client/custom-field';
import {
  DndContext,
  MouseSensor,
  rectIntersection,
  TouchSensor,
  useSensor,
  useSensors
} from '@dnd-kit/core';
import { useMutation, useQuery } from '@tanstack/react-query';
import { Code, Edit, Eye, Loader } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import { toast } from 'sonner';
import FormBuilderCanvas from './FormBuilderCanvas';
import FormBuilderHeader from './FormBuilderHeader';
import FormElementProperties from './FormElementProperties';
import FormElementsCustom from './FormElementsCustom';
import FormPreview from './FormPreview';
import FormPublicUrlDialog from './FormPublicUrlDialog';
import FormSettingsDialog from './FormSettingsDialog';
import FormSuccess from './FormSuccess';
import { useDragHandlers } from './hooks/useDragHandlers';
import { useFormSettings } from './hooks/useFormSettings';
import {
  FormElement,
  FormState
} from './types';
import { getPreviewUrl } from './utils/constants';
import { createCustomForm } from './utils/form-element-utils';
import { convertApiSchemaToZod } from './utils/validation-utils';
import { contactListApi, PayloadFormInBoundRequest } from '@/network/client/contact-list';

const initialFormState = createCustomForm();

export default function FormBuilder({
  formInboundSlug,
  closeSheet,
  onCreateSuccess,
}: {
  formInboundSlug?: string;
  closeSheet: () => void;
  onCreateSuccess?: (formInboundId: string, data: any) => void;
}) {
  const { data: formInbound, isLoading: isLoadingFormInbound } = useQuery({
    queryKey: ['form-inbound-slug', formInboundSlug],
    queryFn: () => contactListApi.getFormInBoundBySlug(formInboundSlug || ''),
    enabled: !!formInboundSlug,
  });

  const { data: customFields, isLoading, refetch } = useQuery({
    queryKey: ['custom-fields'],
    queryFn: () => getCustomFields({ entity_type: 'contact' }),
  });

  const mutationDelete = useMutation({
    mutationFn: (id: string) => deleteCustomField(id),
    onSuccess: () => {
      toast.success('Custom field deleted successfully');
      refetch();
    }
  })

  const mutationCreate = useMutation({
    mutationFn: (data: PayloadFormInBoundRequest) => contactListApi.postFormInBound(data),
    onSuccess: (res) => {
      onCreateSuccess?.(res.data.id, res.data)
      setIsPublicUrlDialogOpen(true)
      toast.success('Form saved successfully');
    },
    onError: (err: any) => {
      const isErrorName = err?.message?.includes('An inbound lead form with slug')
      if (isErrorName) {
        setError(err.message);
      } else {
        toast.error(err.message);
      }
    },
  });

  const mutationUpdate = useMutation({
    mutationFn: (data: PayloadFormInBoundRequest) => contactListApi.putFormInBoundBySlug(formInboundSlug || '', data),
    onSuccess: () => {
      closeSheet();
      setError(null);
      toast.success('Form saved successfully');
    },
    onError: (err: any) => {
      const isErrorName = err?.message?.includes('An inbound lead form with slug')
      if (isErrorName) {
        setError(err.message);
      } else {
        toast.error(err.message);
      }
    },
  });

  const [formState, setFormState] = useState<FormState>(formInboundSlug ? {
    elements: formInbound?.data?.ui_metadata?.elements || [],
    settings: formInbound?.data?.ui_metadata?.settings || {},
  } : initialFormState);
  const [selectedElement, setSelectedElement] = useState<FormElement | null>(null);
  const [activeTab, setActiveTab] = useState('editor');
  const [leftPanelTab, setLeftPanelTab] = useState('components');
  const [jsonOutput, setJsonOutput] = useState(JSON.stringify({}, null, 2));
  const [heightToolbox, setHeightToolbox] = useState('400px');
  const [loadingRegenerateUrl, setLoadingRegenerateUrl] = useState(false);
  const [loadingSuccess, setLoadingSuccess] = useState(false);
  const [isSuccess, setIsSuccess] = useState(false);
  const [submitData, setSubmitData] = useState<any>(null);
  const [error, setError] = useState<string | null>(null);
  const previewUrl = getPreviewUrl(formState);
  const [isPublicUrlDialogOpen, setIsPublicUrlDialogOpen] = useState(false);

  const { activeDragElement, handleDragStart, handleDragOver, handleDragEnd } = useDragHandlers(
    formState,
    setFormState,
    setSelectedElement
  );

  const { currentSettings, isDialogOpen, setCurrentSettings, setIsDialogOpen, handleSettingChange, handleAlignmentChange } = useFormSettings(formState.settings);

  const sensors = useSensors(
    useSensor(MouseSensor, {
      activationConstraint: {
        distance: 5,
      },
    }),
    useSensor(TouchSensor, {
      activationConstraint: {
        delay: 250,
        tolerance: 5,
      },
    })
  );

  const handleFormChange = (
    newFormStateOrUpdater: FormState | ((prev: FormState) => FormState)
  ) => {
    if (typeof newFormStateOrUpdater === 'function') {
      setFormState((prev) => {
        const newState = newFormStateOrUpdater(prev);
        setJsonOutput(JSON.stringify(newState, null, 2));
        return newState;
      });
    } else {
      setFormState(newFormStateOrUpdater);
      setJsonOutput(JSON.stringify(newFormStateOrUpdater, null, 2));
    }
  };

  // Handle element selection with auto-switching to properties tab
  const handleSelectElement = (element: FormElement | null) => {
    if (element) {
      setSelectedElement(element);
      // Auto-switch to properties tab when selecting an element
      setLeftPanelTab('properties');
    } else {
      setSelectedElement(null);
      // Switch back to components tab when deselecting
      setLeftPanelTab('components');
    }
  };

  const handleTabChange = (value: string) => {
    setActiveTab(value);
    if (value === 'code') {
      setJsonOutput(JSON.stringify(formState, null, 2));
    }
  };

  const handleSaveForm = () => {
    const data = {
      name: formState.settings.title,
      slug: formState.settings.publicUrl?.name || '',
      fields: formState.elements.filter(item => !['spacer', 'divider'].includes(item.name)).map((el) => ({
        name: el.name,
        display_name: el.label,
        is_required: el.required,
        field_type: el.type,
        fallback_value: el.defaultValue,
        ui_metadata: {},
      })),
      ui_metadata: formState
    }
    if (formInboundSlug) {
      const cloneData = { ...data } as { slug?: string };
      delete cloneData.slug;
      mutationUpdate.mutate(cloneData as PayloadFormInBoundRequest);
    } else {
      mutationCreate.mutate(data as PayloadFormInBoundRequest);
    }
  };

  useEffect(() => {
    setCurrentSettings(formState.settings);
  }, [formState.settings]);

  useEffect(() => {
    setJsonOutput(JSON.stringify(formState, null, 2));
  }, [formState]);

  // Handler to update form styles from the dialog
  const applySettings = () => {
    setFormState(prevFormState => ({
      ...prevFormState,
      settings: { ...currentSettings },
    }));
    setIsDialogOpen(false);
  };

  // Handler for property changes in the selected element
  const handlePropertyChange = (
    property: keyof FormElement | string,
    value: any
  ) => {
    // Handle form settings properties (like publicUrl)
    if (typeof property === 'string' && property.startsWith('publicUrl.')) {
      const [parentProp, childProp] = property.split('.');
      setFormState(prev => ({
        ...prev,
        settings: {
          ...prev.settings,
          [parentProp]: {
            ...prev.settings?.[parentProp],
            [childProp]: value,
          },
        },
      }));
      return;
    }

    // Handle element properties when an element is selected
    if (!selectedElement) return;

    const newElements = formState.elements.map((el) => {
      if (el.id === selectedElement.id) {
        // For nested properties like options for select/radio
        if (typeof property === 'string' && property.includes('.')) {
          const [parentProp, childProp] = property.split('.');
          return {
            ...el,
            // @ts-ignore - Allow dynamic access for nested properties
            [parentProp]: { ...el[parentProp], [childProp]: value },
          };
        }
        return { ...el, [property]: value };
      }
      return el;
    });

    setFormState(prev => ({ ...prev, elements: newElements }));
    // Update selectedElement state as well to reflect changes immediately in properties panel
    setSelectedElement(prevEl =>
      prevEl ? { ...prevEl, [property]: value } : null
    );
  };

  // Specific handler for style property changes (nested in element.styles)
  const handleStylePropertyChange = (styleProperty: string, value: any) => {
    if (!selectedElement) return;

    const newElements = formState.elements.map((el) => {
      if (el.id === selectedElement.id) {
        const currentStyles = (el as any).styles || {};
        return {
          ...el,
          styles: {
            ...currentStyles,
            [styleProperty]: value,
          },
        };
      }
      return el;
    });

    setFormState(prev => ({ ...prev, elements: newElements }));
    setSelectedElement(prevEl => {
      if (!prevEl) return null;
      const currentStyles = (prevEl as any).styles || {};
      return {
        ...prevEl,
        styles: {
          ...currentStyles,
          [styleProperty]: value,
        },
      };
    });
  };

  const buildPublicUrl = (prevSettings: any) => ({
    ...prevSettings?.publicUrl,
    origin,
    name: `new-customer-form-${generateId()}`,
    path: 'public/forms',
  });

  let timeout1: NodeJS.Timeout;
  let timeout2: NodeJS.Timeout;
  const handleRegenerateUrl = () => {
    setError(null);
    setLoadingRegenerateUrl(true);


    if (timeout1) clearTimeout(timeout1);
    if (timeout2) clearTimeout(timeout2);

    timeout1 = setTimeout(() => {
      setFormState(prev => ({
        ...prev,
        settings: {
          ...prev.settings,
          publicUrl: buildPublicUrl(prev.settings),
        },
      }));
      setLoadingRegenerateUrl(false);
      setLoadingSuccess(true);
      timeout2 = setTimeout(() => {
        setLoadingSuccess(false);
      }, 400);
    }, 500);
  };

  const handleOnSubmit = (data: any) => {
    if (Object.keys(data).length === 0) return;
    setSubmitData(data);
    setIsSuccess(true);
  };

  const handleOnClose = () => {
    setIsSuccess(false);
    setSubmitData(null);
  };

  const handleOnBack = () => {
    setIsSuccess(false);
    setSubmitData(null);
  };

  useEffect(() => {
    if (formInboundSlug) return;
    handleRegenerateUrl();
  }, [formInboundSlug])

  useEffect(() => {
    if (!formInbound || !formInboundSlug) return;

    const mapSelectOptions = (options: string[]) =>
      options.map(option => ({
        label: option,
        value: option.includes('-- Select') ? null : option,
      }));

    const selectOptionsMap = formInbound.data?.fields?.reduce((acc: Record<string, { label: string; value: string | null }[]>, field) => {
      if (field?.select_options?.length) {
        acc[field.name] = mapSelectOptions(field.select_options);
      }
      return acc;
    }, {} as Record<string, { label: string; value: string | null }[]>);

    const processElement = (element: any) => {
      if ((element.type === 'select' || element.type === 'radio') && selectOptionsMap?.[element.name]) {
        return {
          ...element,
          options: selectOptionsMap[element.name],
        };
      }
      return element;
    };

    setFormState({
      elements: formInbound.data?.ui_metadata?.elements?.map(processElement) || [],
      settings: formInbound.data?.ui_metadata?.settings || {},
    });
  }, [formInbound, formInboundSlug]);

  const freezeFields = formInbound?.data?.fields?.filter(item => item.is_required).map(item => item.name) || [];

  const contentRef = useRef<HTMLDivElement>(null);
  const isScrollContent = useIsScroll(contentRef)

  return (
    <DndContext
      sensors={sensors}
      collisionDetection={rectIntersection}
      onDragStart={handleDragStart}
      onDragOver={handleDragOver}
      onDragEnd={handleDragEnd}
    >
      <div className="container ml-auto mr-0 px-4 py-6">
        <FormBuilderHeader
          formState={formState}
          loadingRegenerateUrl={loadingRegenerateUrl}
          loadingSuccess={loadingSuccess}
          onPropertyChange={handlePropertyChange}
          onRegenerateUrl={handleRegenerateUrl}
          onSaveForm={handleSaveForm}
          onOpenSettings={() => setIsDialogOpen(true)}
          error={error}
          closeSheet={closeSheet}
          formInboundSlug={formInboundSlug}
          previewUrl={previewUrl}
        />


        {/* Main three-column layout */}
        <div className={cn("grid grid-cols-1 gap-6 lg:grid-cols-12 h-[calc(100vh-106px)] overflow-y-auto", isScrollContent ? 'border-t' : '')} ref={contentRef}>
          {/* Left column - Components & Properties */}
          <div className="space-y-4 lg:col-span-2">
            <Tabs
              value={leftPanelTab}
              onValueChange={(value) => {
                setLeftPanelTab(value);
                // If switching to properties tab with no element selected, select the first element
                if (
                  value === 'properties' &&
                  !selectedElement &&
                  formState.elements.length > 0
                ) {
                  setSelectedElement(formState.elements[0]);
                }
                if (value === 'components') {
                  setSelectedElement(null);
                }
              }}
            >
              <TabsList className="grid w-full grid-cols-2">
                <TabsTrigger value="components">Components</TabsTrigger>
                <TabsTrigger value="properties">Properties</TabsTrigger>
              </TabsList>
              <TabsContent value="components" className="mt-4">
                <FormElementsCustom
                  onDeleteField={(id) => {
                    mutationDelete.mutate(id);
                  }}
                  onCreateField={() => { refetch() }}
                  formState={formState}
                  customFields={customFields?.data?.list}
                  isLoading={isLoading}
                  setHeightToolbox={setHeightToolbox} />
              </TabsContent>
              <TabsContent value="properties" className="mt-4">
                <FormElementProperties
                  formState={formState}
                  element={selectedElement}
                  propertyExcludedByElement={['spacer', 'divider']}
                  onChange={handlePropertyChange}
                  onStylePropertyChange={handleStylePropertyChange}
                  setHeightToolbox={setHeightToolbox}
                  freezeFields={freezeFields}
                />
              </TabsContent>
            </Tabs>
          </div>

          {/* Middle column - Canvas Builder */}
          <div className="space-y-4 lg:col-span-5">
            <Tabs value={activeTab} onValueChange={handleTabChange}>
              <TabsList className="grid w-full grid-cols-2">
                <TabsTrigger value="editor">
                  <Edit className="mr-2 h-4 w-4" />
                  Edit
                </TabsTrigger>
                <TabsTrigger value="code" disabled={process.env.NODE_ENV !== 'development'}>
                  <Code className="mr-2 h-4 w-4" />
                  JSON
                </TabsTrigger>
              </TabsList>
              <TabsContent value="editor" className="mt-4 space-y-4">
                {isLoadingFormInbound ? (
                  <div className="flex h-[400px] items-center justify-center">
                    <Loader className="h-5 w-5 animate-spin" />
                  </div>
                ) : (
                  <FormBuilderCanvas
                    formState={formState}
                    onChange={handleFormChange}
                    onSelectElement={handleSelectElement}
                    selectedElementId={selectedElement?.id || null}
                    useExternalDndContext={true}
                    draggedOverlayElement={activeDragElement}
                    heightToolbox={heightToolbox}
                  />
                )}
              </TabsContent>
              <TabsContent value="code" className="mt-4">
                <Card>
                  <CardContent className="pt-6">
                    <Textarea
                      className="h-[400px] font-mono"
                      value={jsonOutput}
                      onChange={(e) => setJsonOutput(e.target.value)}
                      readOnly
                    />
                  </CardContent>
                </Card>
              </TabsContent>
            </Tabs>
          </div>

          {/* Right column - Form Preview */}
          <div className="space-y-4 lg:col-span-5">
            <div className="mb-4 flex flex-col gap-2">
              <div className="flex flex-col gap-2">
                <div className="flex items-center gap-2">
                  <Eye className="h-5 w-5" />
                  <h2 className="text-xl font-semibold">Form Preview</h2>
                </div>
              </div>
            </div>
            {
              isLoadingFormInbound ? (
                <div className="flex h-[400px] items-center justify-center">
                  <Loader className="h-5 w-5 animate-spin" />
                </div>
              ) : (
                <>
                  {isSuccess ? (
                    <FormSuccess
                      formState={formState}
                      data={submitData}
                      onBack={handleOnBack}
                      onClose={handleOnClose}
                    />
                  ) : (
                    <FormPreview
                      formState={formState}
                      validationSchema={convertApiSchemaToZod(formState.elements)}
                      onFormSubmit={handleOnSubmit}
                      selectedElement={selectedElement}
                    />
                  )}
                </>
              )
            }
          </div>
        </div>
      </div>

      {/* Settings Dialog */}
      <FormSettingsDialog
        open={isDialogOpen}
        currentSettings={currentSettings}
        onOpenChange={setIsDialogOpen}
        onSettingChange={handleSettingChange}
        onAlignmentChange={handleAlignmentChange}
        onApply={applySettings}
        onCancel={() => {
          setCurrentSettings(formState.settings);
          setIsDialogOpen(false);
        }}
      />
      {/* Public URL Dialog */}
      <FormPublicUrlDialog
        open={isPublicUrlDialogOpen}
        onOpenChange={setIsPublicUrlDialogOpen}
        publicUrl={previewUrl}
        onApply={() => {
          setIsPublicUrlDialogOpen(false);
          window.navigator.clipboard.writeText(previewUrl);
          toast.success('Public URL copied to clipboard!');
          closeSheet();
          setError(null);
        }}
        onCancel={() => setIsPublicUrlDialogOpen(false)}
      />
    </DndContext>
  );
}
