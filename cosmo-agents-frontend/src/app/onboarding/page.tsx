'use client';

import { IconSuccess } from '@/assets/icons';
import { CosmoMark } from '@/components/nav/cosmo-mark';
import { Spinner } from '@/components/ui/spinner';
import OnboardingBackground from '@/images/onboarding_background.png';
import { kyClient } from '@/lib/ky';
import { OnboardingPayload, OnboardingType } from '@/models/auth';
import { ApiResponse } from '@/models/response';
import { extractCompanyInfo } from '@/network/client/ai';
import chatApi from '@/network/client/ai-chat';
import SegmentationApi from '@/network/client/segmentation';
import {
  createOrganization,
  CreateOrganizationData,
} from '@/network/client/organization';
import { useUser } from '@/hooks/use-user';
import {
  Box,
  Button,
  Checkbox,
  Grid,
  Group,
  LoadingOverlay,
  Radio,
  SimpleGrid,
  Stack,
  Stepper,
  TagsInput,
  Text,
  Textarea,
  TextInput,
} from '@mantine/core';
import { useForm } from '@mantine/form';
import { User } from '@sentry/nextjs';
import { useRouter } from 'next/navigation';
import { useEffect, useState } from 'react';
import { toast } from 'sonner';

const LEAD_HANDLING_OPTIONS = [
  'Nurture with marketing automation platforms',
  'Manual follow up by sales team',
  'Just leave them sitting in our CRM',
  'I’m not sure',
];

/**
 * The organization payload plus the ICP answers. The ICP fields are not part of
 * the organization record — they seed the first segmentation so Smart Insights
 * has something to score contacts against from day one.
 */
type OnboardingValues = CreateOrganizationData & {
  icp_industries: string[];
  icp_company_sizes: string[];
  icp_job_titles: string[];
};

/** Trims and adds https:// when the user typed a bare domain. */
function normalizeCompanyUrl(url: string): string {
  const trimmed = (url ?? '').trim();
  if (!trimmed) return '';
  return /^https?:\/\//i.test(trimmed) ? trimmed : `https://${trimmed}`;
}

export default function OnboardingPage() {
  const router = useRouter();
  const { user, isLoading } = useUser();

  useEffect(() => {
    if (user && (user.roles.length > 0 || user.organizations.length > 0)) {
      router.push('/ai-inboxes');
    }
  }, [user]);

  if (isLoading) {
    return <Spinner withOverlay />;
  }

  if (!user) {
    return <div>No user found</div>;
  }

  return <OnboardingForm />;
}

function OnboardingForm() {
  const { user } = useUser();
  const router = useRouter();
  const [active, setActive] = useState<number>(0);
  const [loading, setLoading] = useState<boolean>(false);

  const form = useForm<OnboardingValues>({
    initialValues: {
      name: '',
      company_url: '',
      company_description: '',
      company_targeting_persona: [],
      value_offering: '',
      crm: 'Salesforce',
      lead_handling: [],
      lead_handling_other: '',
      icp_industries: [],
      icp_company_sizes: [],
      icp_job_titles: [],
    },
  });

  const validateStep = (step: number) => {
    const checkFieldError = (
      field: keyof typeof form.values,
      errorMessage: string
    ) => {
      const value = form.values[field];
      // Tag inputs hold an array; everything else is a plain string.
      const isEmpty = Array.isArray(value)
        ? value.length === 0
        : !value?.trim();

      if (isEmpty) {
        form.setFieldError(field, errorMessage);
        return false;
      }
      return true;
    };

    const isValidUrl = (url: string) => {
      const urlPattern =
        /^(https?:\/\/)([\w.-]+)\.[a-zA-Z]{2,}(:\d+)?(\/\S*)?$/i;

      return urlPattern.test(url);
    };

    if (step === 0) {
      // "acme.com" is accepted and stored as "https://acme.com"; validating
      // before adding the scheme rejected every URL typed without one.
      const normalized = normalizeCompanyUrl(form.values.company_url);
      if (normalized !== form.values.company_url) {
        form.setFieldValue('company_url', normalized);
      }
      const isCompanyNameValid = checkFieldError(
        'name',
        'Company name is required'
      );
      const isCompanyUrlValid = !!normalized;
      if (!normalized) {
        form.setFieldError('company_url', 'Company url is required');
      }
      const companyUrl = normalized;

      if (companyUrl && !isValidUrl(companyUrl)) {
        form.setFieldError('company_url', 'Company url must be a valid URL');
        return false;
      }

      return isCompanyNameValid && isCompanyUrlValid;
    }

    if (step === 1) {
      const isDescriptionValid = checkFieldError(
        'company_description',
        'Company description is required'
      );
      const isPersonaValid = checkFieldError(
        'company_targeting_persona',
        'Company targeting persona is required'
      );
      const isOfferingValid = checkFieldError(
        'value_offering',
        'Value offering is required'
      );

      return isDescriptionValid && isPersonaValid && isOfferingValid;
    }

    return true;
  };

  const handleStepClick = async (step: number) => {
    if (
      step <= active ||
      (step === 1 && validateStep(0)) ||
      (step === 2 && validateStep(0) && validateStep(1))
    ) {
      if (active === 0) {
        await handleExtractCompanies(form.values);
      }

      if (active === 1) {
        const isValidStep2 = validateStep(1);
        if (!isValidStep2) {
          return;
        }
      }

      if (active === 2) {
        const isValidStep1 = validateStep(0);
        const isValidStep2 = validateStep(1);
        if (!isValidStep1 || !isValidStep2) {
          return;
        }
      }

      setActive(step);
    }
  };

  // Extraction is not done here: the first step's Next button and the
  // stepper header already run it, and doing it again here called the AI
  // twice and showed any error twice.
  const nextStep = async () => {
    setActive((current) => (current < 3 ? current + 1 : current));
  };

  const prevStep = () =>
    setActive((current) => (current > 0 ? current - 1 : current));

  const handleExtractCompanies = async (values: typeof form.values) => {
    try {
      const companyUrl = normalizeCompanyUrl(values.company_url);
      if (!companyUrl) return;

      const extractData = await extractCompanyInfo(companyUrl);

      if (extractData) {
        form.setFieldValue(
          'company_description',
          extractData.data.company_description || ''
        );
        form.setFieldValue(
          'company_targeting_persona',
          extractData.data.company_targeting_persona || []
        );
        form.setFieldValue(
          'value_offering',
          extractData.data.value_offering || ''
        );
      }
    } catch (err: any) {
      toast.error(
        err.error?.message || err.message || 'Oops! Something went wrong'
      );
    }
  };

  const handleSubmit = async (values: typeof form.values) => {
    setLoading(true);
    try {
      const data: CreateOrganizationData = {
        name: values.name,
        company_url: values.company_url,
        company_description: values.company_description,
        company_targeting_persona: values.company_targeting_persona,
        value_offering: values.value_offering,
        crm: values.crm,
        lead_handling: values.lead_handling,
        // Only meaningful when "Other" is ticked; drop it otherwise so a
        // stale draft does not get persisted.
        lead_handling_other: values.lead_handling.includes('Other')
          ? values.lead_handling_other
          : '',
      };

      await createOrganization(data);
      await createInitialSegment(values);
      await updateOnboarding();
      nextStep();
      toast.success('Submit successfully');
    } catch (err: any) {
      toast.error(
        err.error?.message || err.message || 'Oops! Something went wrong'
      );
    } finally {
      setLoading(false);
    }
  };

  /**
   * Seeds the first segmentation from the ICP answers. A failure here must not
   * fail onboarding — the organization already exists at this point, and the
   * segment can be created later from Smart Insights.
   */
  const createInitialSegment = async (values: OnboardingValues) => {
    const industries = values.icp_industries.filter(Boolean);
    const companySizes = values.icp_company_sizes.filter(Boolean);
    const jobTitles = values.icp_job_titles.filter(Boolean);

    if (
      industries.length === 0 &&
      companySizes.length === 0 &&
      jobTitles.length === 0
    ) {
      return;
    }

    const descriptionParts = [
      industries.length > 0 && `Industries: ${industries.join(', ')}`,
      companySizes.length > 0 && `Company sizes: ${companySizes.join(', ')}`,
      jobTitles.length > 0 && `Job titles: ${jobTitles.join(', ')}`,
    ].filter(Boolean);

    try {
      await SegmentationApi.create({
        name: industries[0] ? `${industries[0]} ICP` : 'Ideal Customer Profile',
        description: descriptionParts.join(' · '),
        priority: 1,
        is_active: true,
        // Keys and array shape are dictated by EvaluateContactFit — it reads
        // ideal_titles / ideal_company_size / ideal_industries and casts each
        // to []any. Anything else scores a flat 50 on every criterion.
        icp_definition: {
          ideal_industries: industries,
          ideal_company_size: companySizes,
          ideal_titles: jobTitles,
        },
      });
    } catch (err) {
      console.error('Failed to create the initial segment:', err);
      toast.warning(
        'Organization created, but the first segment could not be saved. You can add it from Smart Insights.'
      );
    }
  };

  const onCreateConversation = async () => {
    return await chatApi.postCreateConversation();
  };

  const onClearConversation = async (conversationId) => {
    await chatApi.deleteClearConversation(conversationId);
  };

  const extractOnboardingTypeByAI = async () => {
    const knowledge = `${JSON.stringify(form.values)}`;
    const conversationId = await onCreateConversation();
    if (!conversationId) {
      return 'default';
    }
    try {
      const data = await chatApi.postOnboardingAnswer({
        conversationId,
        messages: [
          {
            content: knowledge,
            content_type: 'text',
            role: 'user',
          },
        ],
      });
      await onClearConversation(conversationId);
      return data || 'default';
    } catch (error) {
      console.error('Error during extract onboarding type by AI:', error);
      return 'default';
    }
  };

  const updateOnboarding = async () => {
    const onboardingAnswer = await extractOnboardingTypeByAI();
    const ui_metadata: OnboardingPayload = {
      onboarding: true,
      status: 'not_started',
      step: 0,
      onboarding_type: (onboardingAnswer as OnboardingType) || 'default',
      ai_inboxes: {
        onboarding: true,
        step: 0,
        status: 'not_started',
      },
      campaigns: {
        onboarding: true,
        step: 0,
        status: 'not_started',
      },
      contacts: {
        onboarding: true,
        step: 0,
        status: 'not_started',
      },
      libraries: {
        onboarding: true,
        step: 0,
        status: 'not_started',
      },
      settings: {
        onboarding: true,
        step: 0,
        status: 'not_started',
      },
    };
    try {
      await kyClient
        .patch('v1/users/me', {
          json: {
            ui_metadata,
          },
        })
        .json<ApiResponse<User>>();
    } catch (error) {
      nextStep();
    }
  };

  return (
    <Box h="100vh">
      <Grid gutter={0}>
        <Grid.Col span={6} pos="relative">
          {user ? (
            <Box m="100">
              <Stack h="100%">
                <Group mb="lg">
                  <CosmoMark size={50} />
                  <Text size="xl" fw={600} c="darkblue">
                    COSMO
                  </Text>
                </Group>
                <Text c="gray.5" size="md" fw={600}>
                  Quick survey
                </Text>
                <form
                  onSubmit={form.onSubmit((values) => handleSubmit(values))}
                >
                  <Stepper
                    iconSize={24}
                    active={active}
                    onStepClick={handleStepClick}
                  >
                    <Stepper.Step label="First step">
                      <Text size="xl" fw={600} my="lg">
                        Hi {user.name}, we’d love to learn a bit more about you
                        so we can set up everything you need when you start your
                        journey at Cosmo.
                      </Text>
                      <Box>
                        <TextInput
                          label="Company name"
                          placeholder="Enter company name"
                          key={form.key('name')}
                          {...form.getInputProps('name')}
                          required
                        />
                        <TextInput
                          mt="md"
                          label="Your company URL"
                          placeholder="Enter company url"
                          key={form.key('company_url')}
                          {...form.getInputProps('company_url')}
                          required
                        />
                        <Group justify="end" mt="xl">
                          <Button
                            loading={loading}
                            disabled={loading}
                            onClick={async () => {
                              if (validateStep(0)) {
                                setLoading(true);
                                try {
                                  // form.values still holds the pre-normalised
                                  // URL in this render, so pass it explicitly.
                                  await handleExtractCompanies({
                                    ...form.values,
                                    company_url: normalizeCompanyUrl(
                                      form.values.company_url
                                    ),
                                  });
                                } finally {
                                  setLoading(false);
                                }
                                nextStep();
                              }
                            }}
                          >
                            Next
                          </Button>
                        </Group>
                      </Box>
                    </Stepper.Step>
                    <Stepper.Step label="Second step">
                      <Text size="xl" fw={600} my="lg">
                        Nice, our AI found this information from your website.
                        We will this information to personalize your AI agents.
                      </Text>
                      <Stack>
                        <Textarea
                          label="Company description"
                          placeholder="Hubspot is a company that does etc"
                          key={form.key('company_description')}
                          {...form.getInputProps('company_description')}
                          autosize
                          minRows={1}
                          maxRows={6}
                          required
                        />
                        <TagsInput
                          label="Your company targeting persona"
                          placeholder="Enter tag"
                          defaultValue={[
                            'Marketing Manager',
                            'Business Development Manager',
                          ]}
                          data={[
                            'Digital Transformation Team at Enterprises',
                            'Marketing Manager at Series A Tech Companies',
                            'Marketing Manager at Educational Companies',
                          ]}
                          key={form.key('company_targeting_persona')}
                          {...form.getInputProps('company_targeting_persona')}
                          splitChars={[',']}
                          required
                          clearable
                        />
                        <Textarea
                          label="Your value offering"
                          placeholder="Hubspot is a marketing automation platform that does etc "
                          key={form.key('value_offering')}
                          {...form.getInputProps('value_offering')}
                          autosize
                          minRows={1}
                          maxRows={6}
                          required
                        />
                        <Group justify="end" mt="xl">
                          <Button onClick={prevStep}>Back</Button>
                          <Button
                            onClick={() => {
                              if (validateStep(1)) {
                                nextStep();
                              }
                            }}
                          >
                            Next
                          </Button>
                        </Group>
                      </Stack>
                    </Stepper.Step>
                    <Stepper.Step label="Final step">
                      <Radio.Group
                        {...form.getInputProps('crm')}
                        name="CRM"
                        label="What CRM do you use to manage your leads?"
                        labelProps={{
                          style: {
                            fontSize: '20px',
                            fontWeight: '600',
                            marginBottom: '12px',
                          },
                        }}
                      >
                        <SimpleGrid cols={3} spacing="md">
                          <Radio value="Salesforce" label="Salesforce" />
                          <Radio value="Hubspot" label="Hubspot" />
                          <Radio
                            value="Microsoft Dynamics"
                            label="Microsoft Dynamics"
                          />
                          <Radio value="Others" label="Others" />
                          <Radio
                            value="I'm not using a CRM"
                            label="I'm not using a CRM"
                          />
                          <Radio value="Pipedrive" label="Pipedrive" />
                        </SimpleGrid>
                      </Radio.Group>
                      <Stack>
                        <Text size="xl" fw={600} mt="xl">
                          What are you doing now to convert your inbound and
                          dormant leads into sales pipeline?
                        </Text>
                        <Checkbox.Group
                          {...form.getInputProps('lead_handling')}
                        >
                          <Stack gap="xs">
                            {LEAD_HANDLING_OPTIONS.map((option) => (
                              <Checkbox
                                key={option}
                                value={option}
                                label={option}
                              />
                            ))}
                            <Group align="center">
                              <Checkbox value="Other" label="Other:" />
                              <TextInput
                                placeholder="Enter your current approach"
                                disabled={
                                  !form.values.lead_handling.includes('Other')
                                }
                                {...form.getInputProps('lead_handling_other')}
                              />
                            </Group>
                          </Stack>
                        </Checkbox.Group>
                      </Stack>
                      <Stack mt="xl">
                        <Text size="xl" fw={600}>
                          Who is your ideal customer?
                        </Text>
                        <Text size="sm" c="dimmed">
                          We turn this into your first segment, so the AI can
                          score every contact against it.
                        </Text>
                        <TagsInput
                          label="Industries"
                          placeholder="e.g. B2B SaaS — press Enter to add"
                          {...form.getInputProps('icp_industries')}
                        />
                        <TagsInput
                          label="Company sizes"
                          placeholder="Must match the contact's company_size exactly, e.g. 11-50"
                          {...form.getInputProps('icp_company_sizes')}
                        />
                        <TagsInput
                          label="Job titles you sell to"
                          placeholder="Enter a title and press Enter"
                          {...form.getInputProps('icp_job_titles')}
                        />
                      </Stack>

                      <Group justify="end" mt="xl">
                        <Button onClick={prevStep}>Back</Button>
                        <Button
                          type="submit"
                          loading={loading}
                          disabled={loading}
                        >
                          Submit
                        </Button>
                      </Group>
                    </Stepper.Step>
                    <Stepper.Completed>
                      <Stack align="center" justify="center" h="50vh">
                        <IconSuccess height={50} width={60} />
                        <Text size="xl" fw={600}>
                          Fantastic, let’s get started!
                        </Text>
                        <Button onClick={() => router.push('/ai-inboxes')}>
                          OK
                        </Button>
                      </Stack>
                    </Stepper.Completed>
                  </Stepper>
                </form>
              </Stack>
            </Box>
          ) : (
            <LoadingOverlay visible />
          )}
        </Grid.Col>
        <Grid.Col
          span={6}
          style={{
            height: '100vh',
            backgroundImage: `url(${OnboardingBackground.src})`,
            backgroundSize: 'cover',
            backgroundPosition: 'center',
          }}
        />
      </Grid>
    </Box>
  );
}
