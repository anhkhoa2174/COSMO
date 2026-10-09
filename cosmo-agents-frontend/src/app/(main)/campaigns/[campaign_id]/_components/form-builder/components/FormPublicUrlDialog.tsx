import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import Link from 'next/link';

interface FormPublicUrlDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  publicUrl: string;
  onApply: () => void;
  onCancel: () => void;
}

const FormPublicUrlDialog: React.FC<FormPublicUrlDialogProps> = ({
  open,
  onOpenChange,
  publicUrl,
  onApply,
  onCancel,
}) => {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Public URL</DialogTitle>
          <DialogDescription>
            Copy and share this URL to collect leads
          </DialogDescription>
        </DialogHeader>
        <div className="mt-2 border-t pt-4">
          <div className="mb-3 text-lg font-semibold text-[#4F46E5] underline">
            {
              <Link href={publicUrl} target="_blank">
                {publicUrl}
              </Link>
            }
          </div>
        </div>
        <DialogFooter>
          <DialogClose asChild>
            <Button variant="outline" onClick={onCancel}>
              Cancel
            </Button>
          </DialogClose>
          <Button onClick={onApply}>Copy</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};

export default FormPublicUrlDialog;
