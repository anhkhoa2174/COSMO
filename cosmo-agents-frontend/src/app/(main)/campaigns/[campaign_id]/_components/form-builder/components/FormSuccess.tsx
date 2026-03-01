'use client'
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { FormState } from './types';

interface FormSuccessProps {
    formState?: FormState;
    data: any;
    onBack?: () => void;
    onClose?: () => void;
}

const FormSuccess = ({ formState, data, onBack, onClose }: FormSuccessProps) => {
    return (
        <Card>
            <CardContent>
                <div className="flex flex-col w-full h-full items-center justify-center mt-4">
                    <div className="mb-6 text-center">
                        <div className="inline-block rounded-full bg-green-100 p-3">
                            <svg
                                xmlns="http://www.w3.org/2000/svg"
                                className="h-8 w-8 text-green-500"
                                fill="none"
                                viewBox="0 0 24 24"
                                stroke="currentColor"
                            >
                                <path
                                    strokeLinecap="round"
                                    strokeLinejoin="round"
                                    strokeWidth="2"
                                    d="M5 13l4 4L19 7"
                                ></path>
                            </svg>
                        </div>
                        <h2 className="mt-4 text-xl font-bold">Thank You!</h2>
                        <p className="text-gray-600">
                            We've received your <>{formState?.settings?.title || 'form'}</>.
                        </p>
                    </div>
                    <div className="mb-6 rounded-md bg-gray-50 p-4 w-full min-h-[300px] max-h-[400px] lg:min-w-[300px] lg:max-w-[400px] overflow-y-auto">
                        <h3 className="mb-2 font-medium">Submission Details:</h3>
                        <ul className="text-sm text-gray-700">
                            {formState?.elements?.filter((element) => !['spacer', 'divider'].includes(element.type)).map((element) => {
                                return (
                                    <li key={element.id} className="border-b border-gray-200 py-1 flex justify-between w-full">
                                        <div className="font-medium">{element.label}:</div> <div className='ml-auto'>{data[element.name]}</div>
                                    </li>
                                );
                            })}
                        </ul>
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
export default FormSuccess