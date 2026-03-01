'use client'
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { FormState } from './types';

interface FormErrorProps {
    formState?: FormState;
    data: any;
    onBack?: () => void;
    onClose?: () => void;
}

const FormError = ({ formState, data, onBack, onClose }: FormErrorProps) => {
    return (
        <Card>
            <CardContent>
                <div className="flex flex-col w-full h-full items-center justify-center mt-4">
                    <div className="mb-6 text-center">
                        <div className="inline-block rounded-full bg-red-100 p-3">
                            <svg
                                xmlns="http://www.w3.org/2000/svg"
                                className="h-8 w-8 text-red-500"
                                fill="none"
                                viewBox="0 0 24 24"
                                stroke="currentColor"
                            >
                                <path
                                    strokeLinecap="round"
                                    strokeLinejoin="round"
                                    strokeWidth="2"
                                    d="M6 18L18 6M6 6l12 12"
                                ></path>
                            </svg>
                        </div>
                        <h2 className="mt-4 text-xl font-bold">Error!</h2>
                        <p className="text-gray-600">
                            An error occurred while submitting your <>{formState?.settings?.title || 'form'}</>.
                        </p>
                    </div>
                    <div className="mb-6 rounded-md bg-gray-50 p-4 min-h-[300px] max-h-[400px] lg:min-w-[300px] lg:max-w-[400px]">
                        <h3 className="mb-2 font-medium">Error Details:</h3>
                        <div className="text-sm text-gray-700">
                            {data}
                        </div>
                    </div>
                    <div className="flex gap-2">
                        <Button variant="outline" className='w-[120px]' onClick={onBack}>Back to Form</Button>
                        <Button className="w-[120px] bg-red-600 font-medium text-white hover:bg-red-700" onClick={onClose}>
                            Close
                        </Button>
                    </div>
                </div>
            </CardContent>
        </Card>
    );
}
export default FormError