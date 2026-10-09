"use client";

import FormError from '@/app/(main)/campaigns/[campaign_id]/_components/form-builder/components/FormError';
import FormLoading from '@/app/(main)/campaigns/[campaign_id]/_components/form-builder/components/FormLoading';
import FormPreview from '@/app/(main)/campaigns/[campaign_id]/_components/form-builder/components/FormPreview';
import FormSuccess from '@/app/(main)/campaigns/[campaign_id]/_components/form-builder/components/FormSuccess';
import { FormState } from '@/app/(main)/campaigns/[campaign_id]/_components/form-builder/components/types';
import { convertApiSchemaToZod } from '@/app/(main)/campaigns/[campaign_id]/_components/form-builder/components/utils/validation-utils';
import { contactListApi } from '@/network/client/contact-list';
import Image from 'next/image';
import { useParams, useRouter } from 'next/navigation';
import { useEffect, useState } from 'react';

export default function page() {
  const [formState, setFormState] = useState<FormState>();
  const [loading, setLoading] = useState<boolean>(true);
  const [loadingSubmit, setLoadingSubmit] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const [isSuccess, setIsSuccess] = useState<boolean>(false);
  const [submitData, setSubmitData] = useState<any>(null);

  const router = useRouter()
  const { form_slug } = useParams()

  const handleOnBack = () => {
    setIsSuccess(false)
    setSubmitData(null)
    getFormPublicBySlug();
  }

  const handleOnClose = () => {
    if (typeof window !== 'undefined') {
      window.close();
      router.push('/');
    }
  }

  const getFormPublicBySlug = async () => {
    setLoading(true)
    setError(null)
    if (!form_slug) return;
    try {
      const res = await contactListApi.getPublicFormBySlug(form_slug as string)
      const data = res?.data
      if (!data) return;
      setFormState(data?.ui_metadata || {});
    } catch (error) {
      setError(error?.message)
    } finally {
      setLoading(false)
    }
  }

  const postSubmitForm = async (submitData: any) => {
    if (!form_slug) return;
    setLoadingSubmit(true)
    try {
      const res = await contactListApi.submitFormInBoundBySlug(form_slug as string, submitData)
      const data = res?.data
      if (!data) return;
      setSubmitData(submitData)
      setIsSuccess(true)
    } catch (error) {
      setError(error?.message)
    } finally {
      setLoadingSubmit(false)
    }
  }

  const handleOnSubmit = (data: any) => {
    if (Object.keys(data).length === 0) return;
    postSubmitForm(data)
  };

  useEffect(() => {
    getFormPublicBySlug()
  }, []);

  return (
    <div className='w-full h-screen flex flex-col flex-1 bg-gradient-to-t from-[#2A1166] to-[#ffffff] relative'>
      <div className='w-[40px] h-[40px] absolute top-3 left-3'>
        <Image src='/favicon.svg' width={40} height={40} alt='logo' />
      </div>
      <div className='w-full h-full flex items-center justify-center max-w-[1200px] mx-auto px-2'>
        <div className='relative'>
          <div className='relative w-full lg:min-w-[500px] h-full z-20'>
            {loading ? (
              <FormLoading />
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
                  <>
                    {error ? (
                      <FormError
                        formState={formState}
                        data={error}
                        onBack={handleOnBack}
                        onClose={handleOnClose}
                      />
                    ) : (
                      <FormPreview
                        formState={formState}
                        validationSchema={convertApiSchemaToZod(formState?.elements || [])}
                        onFormSubmit={handleOnSubmit}
                        loading={loadingSubmit}
                      />
                    )}
                  </>
                )}
              </>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
