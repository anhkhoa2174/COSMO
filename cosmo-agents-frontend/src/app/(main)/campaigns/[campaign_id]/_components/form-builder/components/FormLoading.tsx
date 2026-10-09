'use client';
import { Card, CardContent } from '@/components/ui/card';
import { Loader2 } from 'lucide-react';

const FormLoading = () => {
  return (
    <Card>
      <CardContent className="flex max-h-[450px] min-h-[350px] items-center justify-center">
        <Loader2 className="h-[60px] w-[60px] animate-spin" />
      </CardContent>
    </Card>
  );
};
export default FormLoading;
