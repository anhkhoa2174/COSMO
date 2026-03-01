import { CreateCustomFieldForm } from '@/components/forms/create-custom-fields-form';
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogHeader,
    DialogTitle,
    DialogTrigger
} from '@/components/ui/dialog';
import React from 'react';

interface FormCreateCustomFieldDialogProps {
    onSave: () => void;
    children: React.ReactNode;
}

const FormCreateCustomFieldDialog: React.FC<FormCreateCustomFieldDialogProps> = ({
    onSave,
    children
}) => {
    const [openDialog, setOpenDialog] = React.useState(false);
    const handleSave = () => {
        setOpenDialog(false);
        onSave();
    };
    return (
        <Dialog open={openDialog} onOpenChange={setOpenDialog}>
            <DialogTrigger asChild>
                {children}
            </DialogTrigger>
            <DialogContent>
                <DialogHeader>
                    <DialogTitle>Add field</DialogTitle>
                    <DialogDescription>
                        Add a new field to your contact or company entity.
                    </DialogDescription>
                </DialogHeader>
                <CreateCustomFieldForm
                    isFormBuilder
                    onSuccess={handleSave}
                />
            </DialogContent>
        </Dialog>
    );
};

export default FormCreateCustomFieldDialog;
