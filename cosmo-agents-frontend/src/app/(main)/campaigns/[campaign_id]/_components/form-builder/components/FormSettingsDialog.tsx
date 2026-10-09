import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Textarea } from '@/components/ui/textarea';
import { cn } from '@/lib/utils';
import type { FormSettings } from './types';

interface FormSettingsDialogProps {
  open: boolean;
  currentSettings: FormSettings;
  onOpenChange: (open: boolean) => void;
  onSettingChange: (
    e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>,
    checkedValue?: boolean
  ) => void;
  onAlignmentChange: (value: string) => void;
  onApply: () => void;
  onCancel: () => void;
}

const FormSettingsDialog: React.FC<FormSettingsDialogProps> = ({
  open,
  currentSettings,
  onOpenChange,
  onSettingChange,
  onAlignmentChange,
  onApply,
  onCancel,
}) => {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Form Settings</DialogTitle>
          <DialogDescription>
            Customize your form's appearance and details
          </DialogDescription>
        </DialogHeader>
        <div
          className={cn(
            'grid max-h-[70vh] grid-cols-1 gap-x-6 gap-y-4 overflow-y-auto border-t py-4 pr-2 md:grid-cols-2'
          )}
        >
          <div className="md:col-span-2">
            <h4 className="text-md mb-2 font-semibold">Form Details</h4>
          </div>
          <div className="md:col-span-2">
            <Label htmlFor="title">Form Title</Label>
            <Input
              id="title"
              name="title"
              value={currentSettings?.title}
              onChange={onSettingChange}
            />
          </div>
          <div className="md:col-span-2">
            <Label htmlFor="description">Form Description</Label>
            <Textarea
              id="description"
              name="description"
              value={currentSettings?.description || ''}
              onChange={onSettingChange}
            />
          </div>
          <div className="mt-2 border-t pt-4 md:col-span-2">
            <h4 className="text-md mb-3 font-semibold">
              Submit Button Appearance
            </h4>
          </div>
          <div>
            <Label htmlFor="submitButtonText">Button Text</Label>
            <Input
              id="submitButtonText"
              name="submitButtonText"
              value={currentSettings?.submitButtonText}
              onChange={onSettingChange}
            />
          </div>
          <div>
            <Label htmlFor="sba_backgroundColor">Button Background Color</Label>
            <div className="relative flex gap-2">
              <Input
                id="sba_backgroundColor"
                name="sba_backgroundColor"
                value={currentSettings?.submitButtonAppearance?.backgroundColor}
                onChange={onSettingChange}
              />
              <input
                id="sba_backgroundColor1"
                name="sba_backgroundColor1"
                className="opacity-1 absolute right-2 top-1/2 h-6 w-6 -translate-y-1/2 cursor-pointer"
                type="color"
                value={currentSettings?.submitButtonAppearance?.backgroundColor}
                onChange={onSettingChange}
              />
            </div>
          </div>
          <div>
            <Label htmlFor="sba_textColor">Button Text Color</Label>
            <div className="relative flex gap-2">
              <Input
                id="sba_textColor"
                name="sba_textColor"
                value={currentSettings?.submitButtonAppearance?.textColor}
                onChange={onSettingChange}
              />
              <input
                id="sba_textColor1"
                name="sba_textColor1"
                className="opacity-1 absolute right-2 top-1/2 h-6 w-6 -translate-y-1/2 cursor-pointer"
                type="color"
                value={currentSettings?.submitButtonAppearance?.textColor}
                onChange={onSettingChange}
              />
            </div>
          </div>
          <div>
            <Label htmlFor="sba_alignment">Button Alignment</Label>
            <Select
              name="sba_alignment"
              value={currentSettings?.submitButtonAppearance?.alignment}
              onValueChange={onAlignmentChange}
            >
              <SelectTrigger>
                <SelectValue placeholder="Select alignment" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="left">Left</SelectItem>
                <SelectItem value="center">Center</SelectItem>
                <SelectItem value="right">Right</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="flex items-center space-x-2 pt-1 md:col-span-2">
            <Checkbox
              id="sba_fullWidth"
              name="sba_fullWidth"
              checked={currentSettings?.submitButtonAppearance?.fullWidth}
              onCheckedChange={(checked) => {
                onSettingChange(
                  { target: { name: 'sba_fullWidth', value: '' } } as any,
                  checked as boolean
                );
              }}
            />
            <Label htmlFor="sba_fullWidth" className="font-normal">
              Full Width Button
            </Label>
          </div>
          <div className="mt-2 border-t pt-4 md:col-span-2">
            <h4 className="text-md mb-3 font-semibold">General Form Styles</h4>
          </div>
          <div>
            <Label htmlFor="margin">Margin</Label>
            <Input
              id="margin"
              name="margin"
              value={currentSettings?.styles?.margin}
              onChange={onSettingChange}
            />
          </div>
          <div>
            <Label htmlFor="padding">Padding</Label>
            <Input
              id="padding"
              name="padding"
              value={currentSettings?.styles?.padding}
              onChange={onSettingChange}
            />
          </div>
          <div>
            <Label htmlFor="backgroundColor">Background Color (Form)</Label>
            <div className="relative flex gap-2">
              <Input
                id="backgroundColor"
                name="backgroundColor"
                value={currentSettings?.styles?.backgroundColor}
                onChange={onSettingChange}
              />
              <input
                id="backgroundColor1"
                name="backgroundColor1"
                className="opacity-1 absolute right-2 top-1/2 h-6 w-6 -translate-y-1/2 cursor-pointer"
                type="color"
                value={currentSettings?.styles?.backgroundColor}
                onChange={onSettingChange}
              />
            </div>
          </div>
          <div>
            <Label htmlFor="gap">Gap (between elements)</Label>
            <Input
              id="gap"
              name="gap"
              value={currentSettings?.styles?.gap}
              onChange={onSettingChange}
            />
          </div>
          <div>
            <Label htmlFor="borderRadius">Border Radius</Label>
            <Input
              id="borderRadius"
              name="borderRadius"
              value={currentSettings?.styles?.borderRadius || ''}
              onChange={onSettingChange}
            />
          </div>
          <div>
            <Label htmlFor="border">Border</Label>
            <Input
              id="border"
              name="border"
              value={currentSettings?.styles?.border || ''}
              onChange={onSettingChange}
            />
          </div>
          <div>
            <Label htmlFor="boxShadow">Box Shadow</Label>
            <Input
              id="boxShadow"
              name="boxShadow"
              value={currentSettings?.styles?.boxShadow || ''}
              onChange={onSettingChange}
            />
          </div>
          <div>
            <Label htmlFor="gridColumns">Grid Columns</Label>
            <Input
              min={1}
              max={4}
              type="number"
              id="gridColumns"
              name="gridColumns"
              value={currentSettings?.gridColumns || ''}
              onChange={onSettingChange}
            />
          </div>
        </div>
        <DialogFooter>
          <DialogClose asChild>
            <Button variant="outline" onClick={onCancel}>
              Cancel
            </Button>
          </DialogClose>
          <Button onClick={onApply}>Apply Changes</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};

export default FormSettingsDialog;
