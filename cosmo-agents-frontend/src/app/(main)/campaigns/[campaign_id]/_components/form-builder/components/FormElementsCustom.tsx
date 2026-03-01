'use client';

import { MainButton } from '@/components/buttons/main-button';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Label } from '@/components/ui/label';
import { CustomField } from '@/models/custom-field';
import { useDraggable } from '@dnd-kit/core';
import { CSS } from '@dnd-kit/utilities';
import { Skeleton } from '@mantine/core';
import {
    ArrowDownUp,
    Briefcase,
    Building2,
    Mail,
    MapPin,
    Minus,
    Phone,
    Plus,
    Trash2,
    User
} from 'lucide-react';
import React, { useEffect, useMemo, useRef } from 'react';
import { v4 as uuidv4 } from 'uuid';
import { getElementIcon } from './FormBuilderCanvas';
import FormCreateCustomFieldDialog from './FormCreateCustomFieldDialog';
import { FormElement, FormElementType, FormState } from './types';

export const systemFields = [
    {
        normalized_name: 'first_name',
        name: 'First name',
        data_type: 'text',
        is_required: true,
        icon: <User size={18} />,
    },
    {
        normalized_name: 'last_name',
        name: 'Last name',
        data_type: 'text',
        is_required: true,
        icon: <User size={18} />,
    },
    {
        normalized_name: 'email',
        name: 'Email',
        data_type: 'email',
        is_required: true,
        icon: <Mail size={18} />,
    },
    {
        normalized_name: 'phone',
        name: 'Phone',
        data_type: 'text',
        is_required: false,
        icon: <Phone size={18} />,
    },
    {
        normalized_name: 'company',
        name: 'Company',
        data_type: 'text',
        is_required: false,
        icon: <Building2 size={18} />,
    },
    {
        normalized_name: 'job_title',
        name: 'Job title',
        data_type: 'text',
        is_required: false,
        icon: <Briefcase size={18} />,
    },
    {
        normalized_name: 'address',
        name: 'Address',
        data_type: 'text',
        is_required: false,
        icon: <MapPin size={18} />,
    },
    {
        normalized_name: 'city',
        name: 'City',
        data_type: 'text',
        is_required: false,
        icon: <MapPin size={18} />,
    },
    {
        normalized_name: 'country',
        name: 'Country',
        data_type: 'text',
        is_required: false,
        icon: <MapPin size={18} />,
    },
    {
        normalized_name: 'state',
        name: 'State',
        data_type: 'text',
        is_required: false,
        icon: <MapPin size={18} />,
    },
    {
        normalized_name: 'zip',
        name: 'Zip',
        data_type: 'text',
        is_required: false,
        icon: <MapPin size={18} />,
    },
];

interface ElementButtonProps {
    id?: string;
    typeItem: string;
    name: string;
    label: string;
    icon: React.ReactNode;
    type?: FormElementType;
    required?: boolean;
    options?: { label: string; value: string }[];
    defaultValue?: string;
    onDelete?: () => void;
}

const DraggableElementButton = ({
    id,
    name,
    type,
    typeItem = 'toolbox-item',
    label,
    icon,
    required,
    options,
    defaultValue,
    onDelete,
}: ElementButtonProps) => {
    const { attributes, listeners, setNodeRef, transform, isDragging } =
        useDraggable({
            id: `toolbox-${name}`,
            data: {
                id: id || uuidv4(),
                type: typeItem,
                name: name,
                label: label,
                elementType: type,
                required: required,
                options: options,
                defaultValue: defaultValue,
            },
        });

    const style = transform
        ? {
            transform: CSS.Transform.toString(transform),
            zIndex: isDragging ? 1000 : 1,
            opacity: isDragging ? 0 : 1,
            boxShadow: isDragging ? '0 5px 15px rgba(0, 0, 0, 0.15)' : 'none',
        }
        : {
            opacity: isDragging ? 0 : 1,
            zIndex: isDragging ? 1000 : 1,
        };

    return (
        <div
            ref={setNodeRef}
            {...listeners}
            {...attributes}
            className="cursor-grab"
            style={style}
        >
            {onDelete ? (
                <div className="mb-2 flex items-center justify-between rounded-md border bg-background p-3 transition-colors hover:bg-accent hover:text-accent-foreground group">
                    <div className="flex items-center space-x-2">
                        <div className="h-5 w-5">{icon}</div>
                        <span>{label}</span>
                    </div>
                    <div className="z-30 flex items-center space-x-1 opacity-0 transition-opacity duration-200 group-hover:opacity-100">
                        <Button
                            variant="ghost"
                            size="icon"
                            className="h-6 w-6 text-destructive"
                            onClick={(e) => {
                                e.stopPropagation();
                                onDelete?.();
                            }}
                        >
                            <Trash2 size={14} />
                        </Button>
                    </div>
                </div>
            ) : (
                <div className="mb-2 flex items-center rounded-md border bg-background p-3 transition-colors hover:bg-accent hover:text-accent-foreground">
                    <div className="mr-2 h-5 w-5">{icon}</div>
                    <span>{label}</span>
                </div>
            )}
        </div>
    );
};

interface FormElementsToolboxProps {
    formState: FormState;
    isLoading?: boolean;
    customFields?: CustomField[];
    setHeightToolbox: (height: string) => void;
    onCreateField: () => void;
    onDeleteField?: (id: string) => void;
}

export default function FormElementsToolbox({
    formState,
    isLoading,
    customFields,
    setHeightToolbox,
    onCreateField,
    onDeleteField
}: FormElementsToolboxProps) {
    const heightComponent = useRef<HTMLDivElement>(null);

    const usedFieldNames = useMemo(() => {
        return new Set(formState.elements.map((el) => el.name).filter(Boolean));
    }, [formState.elements]);

    const dataSystemFields = (systemFields || []).map((field) => ({
        id: uuidv4(),
        type: field.data_type as FormElementType,
        name: field.normalized_name,
        label: field.name,
        icon: field.icon,
        required: field.is_required,
        typeItem: 'toolbox-item',
    }));

    const dataCustomFields = useMemo(() => {
        return (
            (customFields?.map((field) => ({
                id: field.id,
                type: field?.data_type as FormElementType,
                name: field?.normalized_name,
                label: field?.name,
                required: field.is_required,
                icon: getElementIcon({ type: field.data_type } as FormElement),
                options: field.options.map((option) => ({
                    label: option,
                    value: option,
                })),
                defaultValue: field?.fallback_value || null,
                typeItem: 'toolbox-item',
            })) as ElementButtonProps[]) || []
        );
    }, [customFields]);

    const finalCustomFields = useMemo(() => {
        return dataCustomFields.filter(
            (element) => !usedFieldNames.has(element.name)
        );
    }, [dataCustomFields, usedFieldNames]);

    const finalSystemFields = useMemo(() => {
        return dataSystemFields.filter(
            (element) => !usedFieldNames.has(element.name)
        );
    }, [dataSystemFields, usedFieldNames]);

    const finalLayoutFields = useMemo(() => {
        return [
            {
                id: uuidv4(),
                type: 'spacer',
                label: 'Spacer',
                name: 'spacer',
                icon: <ArrowDownUp size={18} />,
                typeItem: 'toolbox-item',
            },
            {
                id: uuidv4(),
                type: 'divider',
                label: 'Divider',
                name: 'divider',
                icon: <Minus size={18} />,
                typeItem: 'toolbox-item',
            },
        ];
    }, []);

    useEffect(() => {
        if (heightComponent.current) {
            const height = heightComponent.current.offsetHeight;
            if (setHeightToolbox) {
                setHeightToolbox(`${height}px`);
            }
        }
    }, [customFields?.length]);

    return (
        <Card ref={heightComponent} className="h-fit w-full">
            <CardHeader className="rounded-t-lg border-b bg-[#F1F5F9] p-0 px-4 pb-3 pt-4">
                <CardTitle>
                    <span className="capitalize text-[#071F1D]">Elements</span>
                </CardTitle>
            </CardHeader>
            <CardContent className="p-2">
                <div className="space-y-1">
                    <Label>Custom Fields</Label>
                    <div className="w-full">
                        <FormCreateCustomFieldDialog onSave={onCreateField}>
                            <MainButton className='w-full h-[42px] !mb-1 border-dashed border-dashed-2 opacity-100' variant="outline" icon={Plus} text="Add Field" />
                        </FormCreateCustomFieldDialog>
                    </div>
                    {isLoading ? (
                        <div className="flex w-full flex-col space-y-1">
                            {Array.from({ length: 3 }).map((_, index) => (
                                <Skeleton
                                    key={index}
                                    className="h-[47px] w-full rounded-md border"
                                />
                            ))}
                        </div>
                    ) : finalCustomFields.length > 0 ? (
                        finalCustomFields.map((element: any) => {
                            return (
                                <DraggableElementButton
                                    id={element?.id}
                                    key={element?.id}
                                    name={element?.name}
                                    typeItem={element.typeItem}
                                    type={element?.type}
                                    required={element?.required}
                                    label={element?.label}
                                    icon={element?.icon}
                                    options={element?.options}
                                    defaultValue={element?.defaultValue || null}
                                    onDelete={() => onDeleteField?.(element?.id)}
                                />
                            );
                        })
                    ) : (
                        <div className="py-4 text-center text-sm text-muted-foreground">
                            No available custom fields
                        </div>
                    )}
                    <Label>System Fields</Label>
                    {finalSystemFields.length > 0 ? (
                        finalSystemFields.map((element: any) => {
                            return (
                                <DraggableElementButton
                                    id={element?.id}
                                    key={element?.id}
                                    name={element?.name}
                                    typeItem={element.typeItem}
                                    type={element?.type}
                                    required={element?.required}
                                    label={element?.label}
                                    icon={element?.icon}
                                    defaultValue={element?.defaultValue || null}
                                    options={element?.options}
                                />
                            );
                        })
                    ) : (
                        <div className="py-4 text-center text-sm text-muted-foreground">
                            No available system fields
                        </div>
                    )}
                    <Label>Layout Fields (Reusable)</Label>
                    {
                        finalLayoutFields.map((element: any) => {
                            return (
                                <DraggableElementButton
                                    key={element?.name + element?.type}
                                    name={element?.name}
                                    typeItem={element.typeItem}
                                    type={element?.type}
                                    required={element?.required}
                                    label={element?.label}
                                    icon={element?.icon}
                                />
                            );
                        })
                    }
                </div>
            </CardContent>
        </Card>
    );
}
