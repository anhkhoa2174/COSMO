'use client';

import { IconCSV, IconHubspot } from '@/assets/icons';
import { AddButton } from '@/components/buttons/add-button';
import { DeleteButton } from '@/components/buttons/delete-button';
import { MainButton } from '@/components/buttons/main-button';
import { DataTable, type ColumnDef } from '@/components/data-table/data-table';
import { CreateContactDialog } from '@/components/forms/create-contact-form';
import { ImportFromHubSpotDialog } from '@/components/forms/import-hubspot-dialog';
import { ContentLayout } from '@/components/nav/content-layout';
import DescriptionText from '@/components/tour/DescriptionText';
import { Badge } from '@/components/ui/badge';
import { ContactExpandedRow } from '@/components/intelligence/contact-expanded-row';
import { ContactStatusBadge } from '@/components/contacts/contact-status-badge';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  createTourCustomContent,
  UpdateOnboarding,
  useTour,
} from '@/hooks/use-tour';
import type { Contact } from '@/models/contact';
import ContactApi from '@/network/client/contact';
import { contactListApi } from '@/network/client/contact-list';
import { getCustomFields } from '@/network/client/custom-field';
import SegmentationApi from '@/network/client/segmentation';
import { useUser } from '@/hooks/use-user';
import {
  currentPageAtom,
  isChangeContact,
  selectedIdAtom,
} from '@/stores/atom';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useAtom } from 'jotai';
import {
  CloudUpload,
  Edit2,
  MoreVertical,
  Trash2,
  Star,
  Filter,
  RefreshCw,
  Users,
} from 'lucide-react';
import { PageHero } from '@/components/ui/page-hero';
import {
  isProfileUrl,
  resolveContactInfo,
  shortenProfileUrl,
} from '@/lib/contact-info';
import { DropdownMenuSeparator } from '@/components/ui/dropdown-menu';
import { useRouter } from 'next/navigation';
import { FormEvent, useEffect, useState, type PropsWithChildren } from 'react';
import { toast } from 'sonner';

export default function ContactsPage() {
  const router = useRouter();
  const { user, invalidate } = useUser();
  // Check if user is admin - admin can see all contacts and "Added By" column
  const isAdmin = user?.roles?.some((r) => r.name === 'admin') ?? false;
  const [filter, setFilter] = useState<Record<any, any>>({});
  const [, setIsChange] = useAtom(isChangeContact);
  const [currentPage] = useAtom(currentPageAtom);
  const pageSize = 100;
  const [selectedId, setSelectedId] = useAtom(selectedIdAtom);
  const [selectedRow, setSelectedRow] = useState<Contact | null>(null);
  const [priorityFilter, setPriorityFilter] = useState(false);
  const [selectedSegmentId, setSelectedSegmentId] = useState<string>('__all__');
  const [selectedStatusFilter, setSelectedStatusFilter] =
    useState<string>('__all__');
  const pageKey = 'contactsTable';

  const {
    data: contacts,
    isLoading,
    refetch: refetchContacts,
    error,
  } = useQuery({
    queryKey: [
      'contacts',
      { filter },
      { offset: ((currentPage[pageKey] || 1) - 1) * pageSize, limit: pageSize },
    ],
    queryFn: () =>
      ContactApi.list(
        { filter },
        {
          offset: ((currentPage[pageKey] || 1) - 1) * pageSize,
          limit: pageSize,
        }
      ),
  });

  const { data: fieldData, refetch: refetchFieldsValue } = useQuery({
    queryKey: ['contacts', 'fieldValue'],
    queryFn: () =>
      ContactApi.getValue({
        fields: ['company', 'job_title'],
        offset: 0,
        limit: 100,
      }),
  });
  const companyValue =
    fieldData?.data?.list?.find((item) => item.company)?.company?.list || [];
  const jobTitleValue =
    fieldData?.data?.list?.find((item) => item.job_title)?.job_title?.list ||
    [];
  const _fieldsValue = {
    company: companyValue.filter(
      (item: unknown) =>
        !((typeof item === 'string' && !item.length) || item === null)
    ),
    job_title: jobTitleValue.filter(
      (item: unknown) =>
        !((typeof item === 'string' && !item.length) || item === null)
    ),
  };

  const { data: customFields } = useQuery({
    queryKey: ['customFields'],
    queryFn: getCustomFields,
  });

  const { data: segments } = useQuery({
    queryKey: ['segmentations'],
    queryFn: () => SegmentationApi.list(true),
  });

  const createSuccess = () => {
    refetchContacts();
    setIsChange((prevIsChange: any) => !prevIsChange);
    refetchFieldsValue();
  };

  const handleDelete = async (contact_ids: string[]) => {
    const deletePromise = ContactApi.delete({ ids: contact_ids });

    toast.promise(deletePromise, {
      loading: 'Deleting...',
      success: () => {
        setSelectedId({});
        refetchContacts();
        refetchFieldsValue();
        return `${contact_ids.length} ${contact_ids.length > 1 ? 'contacts' : 'contact'} has been deleted`;
      },
      error: 'Failed to delete contacts',
    });
  };

  const recalculateStatusMutation = useMutation({
    mutationFn: ContactApi.recalculateStatus,
    onSuccess: (data) => {
      toast.success(`Updated ${data.data?.updated_count || 0} contacts`);
      refetchContacts();
    },
    onError: () => {
      toast.error('Failed to recalculate statuses');
    },
  });

  // Calculate priority score for each contact
  const calculatePriorityScore = (contact: Contact): number => {
    let score = 0;

    // High-priority custom fields (meeting scheduled, etc.)
    const customFields = contact.profile?.custom_fields || {};
    const hasUpcomingMeeting = Object.keys(customFields).some(
      (key) =>
        key.toLowerCase().includes('meeting') ||
        key.toLowerCase().includes('next_call')
    );
    if (hasUpcomingMeeting) score += 100;

    // Buying signals strength
    const buyingSignals = contact.ai_insights?.buying_signals || [];
    buyingSignals.forEach((signal) => {
      if (signal.strength === 'high') score += 30;
      else if (signal.strength === 'medium') score += 15;
      else score += 5;

      // Recency bonus
      if (signal.recency_score && signal.recency_score > 80) score += 20;
    });

    // Pain points and goals count (shows engagement/research depth)
    const painPoints = contact.ai_insights?.suspected_pain_points || [];
    const goals = contact.ai_insights?.suspected_goals || [];
    score += Math.min(painPoints.length * 10, 30); // Max 30 points
    score += Math.min(goals.length * 10, 30); // Max 30 points

    // Research findings indicate active prospect
    const researchFindings = contact.profile?.research_findings || [];
    score += Math.min(researchFindings.length * 5, 25); // Max 25 points

    // High confidence insights
    painPoints.forEach((p) => {
      if (p.confidence && p.confidence > 80) score += 10;
    });
    goals.forEach((g) => {
      if (g.confidence && g.confidence > 80) score += 10;
    });

    return score;
  };

  let rawData =
    contacts?.data?.list.flatMap((item: { entity: any }) => item.entity) || [];

  // Apply priority filter if enabled
  if (priorityFilter && rawData.length > 0) {
    const contactsWithScores = rawData.map((contact) => ({
      contact,
      score: calculatePriorityScore(contact),
    }));

    // Sort by score descending
    contactsWithScores.sort((a, b) => b.score - a.score);

    // Filter contacts with score > 0 (has at least some priority indicators)
    rawData = contactsWithScores
      .filter((item) => item.score > 0)
      .map((item) => item.contact);
  }

  const data = rawData;
  const pageCount = contacts?.data?.total || 0;

  const columns: ColumnDef<Contact>[] = [
    {
      accessorKey: 'index',
      header: '#',
      cell: ({ row }) => {
        const index = data.findIndex((item) => item.id === row.id);
        const page = currentPage[pageKey] || 1;
        return (
          <span className="text-muted-foreground">
            {(page - 1) * pageSize + index + 1}
          </span>
        );
      },
    },
    {
      accessorKey: 'name',
      sortable: true,
      header: 'Name',
      cell: ({ row }) => {
        const score = calculatePriorityScore(row);
        return (
          <div className="flex items-center gap-2">
            <span className="font-semibold">{row.name || '-'}</span>
            {priorityFilter && score > 0 && (
              <Badge
                variant="secondary"
                className={`text-xs ${
                  score >= 100
                    ? 'border-red-300 bg-red-100 text-red-700'
                    : score >= 50
                      ? 'border-orange-300 bg-orange-100 text-orange-700'
                      : 'border-yellow-300 bg-yellow-100 text-yellow-700'
                }`}
              >
                <Star className="mr-1 inline h-3 w-3" />
                {score}
              </Badge>
            )}
          </div>
        );
      },
    },
    {
      accessorKey: 'source',
      header: 'Source',
      cell: ({ row }) => {
        const source = row.source || 'manual';
        const sourceColors: Record<string, string> = {
          linkedin_extension: 'bg-blue-100 text-blue-700',
          linkedin_connections: 'bg-blue-100 text-blue-700',
          apollo: 'bg-purple-100 text-purple-700',
          manual: 'bg-gray-100 text-gray-700',
        };
        const sourceLabels: Record<string, string> = {
          linkedin_extension: 'LinkedIn',
          linkedin_connections: 'LinkedIn',
          apollo: 'Apollo',
          manual: 'Manual',
        };
        return (
          <Badge
            variant="secondary"
            className={sourceColors[source] || 'bg-gray-100 text-gray-700'}
          >
            {sourceLabels[source] || source}
          </Badge>
        );
      },
    },
    {
      accessorKey: 'contact_information',
      header: 'Contact Info',
      cell: ({ row }) => {
        const contactInfo = resolveContactInfo(row);

        if (!contactInfo) {
          return <span className="text-muted-foreground">-</span>;
        }

        // Decided by the value, not by row.source — a LinkedIn-sourced contact
        // may carry an email, and an email-sourced one may carry a URL.
        if (isProfileUrl(contactInfo)) {
          return (
            <a
              href={
                contactInfo.startsWith('http')
                  ? contactInfo
                  : `https://${contactInfo}`
              }
              target="_blank"
              rel="noopener noreferrer"
              className="block max-w-[200px] truncate text-sm text-blue-600 hover:underline"
              title={contactInfo}
            >
              {shortenProfileUrl(contactInfo)}
            </a>
          );
        }

        return (
          <span
            className="block max-w-[220px] truncate text-sm"
            title={contactInfo}
          >
            {contactInfo}
          </span>
        );
      },
    },
    {
      accessorKey: 'job_title',
      sortable: true,
      header: 'Job title',
    },
    {
      accessorKey: 'created_at',
      sortable: true,
      header: 'Created At',
      cell: ({ row }) => {
        const createdAt = row.created_at;
        if (!createdAt) return <span className="text-muted-foreground">-</span>;
        return (
          <span className="text-sm">
            {new Date(createdAt).toLocaleDateString('en-US', {
              year: 'numeric',
              month: 'short',
              day: 'numeric',
            })}
          </span>
        );
      },
    },
    {
      accessorKey: 'status',
      header: 'Status',
      cell: ({ row }) => (
        <ContactStatusBadge
          status={row.status}
          missingFields={row.missing_fields}
        />
      ),
    },
    {
      accessorKey: 'added_by_name',
      header: 'Added By',
      cell: ({ row }) => {
        const name = row.added_by_name;
        const email = row.added_by_email;
        if (!name && !email)
          return <span className="text-muted-foreground">-</span>;
        return (
          <div className="flex flex-col">
            <span className="text-sm font-medium">{name || '-'}</span>
            {email && (
              <span className="text-xs text-muted-foreground">{email}</span>
            )}
          </div>
        );
      },
    },
    {
      accessorKey: 'company',
      sortable: true,
      header: 'Company',
    },
    {
      accessorKey: 'lifecycle_stage',
      header: 'Stage',
      cell: ({ row }) => {
        const stage = row.lifecycle_stage || 'new';
        const stageColors: Record<string, string> = {
          new: 'bg-gray-100 text-gray-700',
          contacted: 'bg-blue-100 text-blue-700',
          replied: 'bg-green-100 text-green-700',
          qualified: 'bg-purple-100 text-purple-700',
          proposal: 'bg-orange-100 text-orange-700',
          won: 'bg-emerald-100 text-emerald-700',
          lost: 'bg-red-100 text-red-700',
        };
        return (
          <Badge
            variant="secondary"
            className={`capitalize ${stageColors[stage] || ''}`}
          >
            {stage}
          </Badge>
        );
      },
    },
    {
      accessorKey: 'outreach_decision',
      header: 'Decision',
      cell: ({ row }) => {
        const decision = row.outreach_decision || 'INTRO';
        const decisionColors: Record<string, string> = {
          INTRO: 'bg-blue-100 text-blue-700',
          'FOLLOW-UP': 'bg-purple-100 text-purple-700',
          NURTURE: 'bg-orange-100 text-orange-700',
          HOLD: 'bg-gray-100 text-gray-700',
        };
        return (
          <Badge variant="secondary" className={decisionColors[decision] || ''}>
            {decision}
          </Badge>
        );
      },
    },
    {
      accessorKey: 'next_step',
      header: 'Next Step',
      cell: ({ row }) => {
        const step = row.next_step || 'SEND';
        const stepColors: Record<string, string> = {
          SEND: 'bg-blue-100 text-blue-700',
          FOLLOW_UP_1: 'bg-yellow-100 text-yellow-700',
          FOLLOW_UP_2: 'bg-orange-100 text-orange-700',
          SET_MEETING: 'bg-purple-100 text-purple-700',
          FOLLOW_UP_MEETING_1: 'bg-indigo-100 text-indigo-700',
          FOLLOW_UP_MEETING_2: 'bg-violet-100 text-violet-700',
          PREPARE_MEETING: 'bg-green-100 text-green-700',
          WAIT: 'bg-gray-100 text-gray-700',
          FOLLOW_UP: 'bg-teal-100 text-teal-700',
          DROP: 'bg-red-100 text-red-700',
        };
        const stepLabels: Record<string, string> = {
          SEND: 'Send',
          FOLLOW_UP_1: 'FU #1',
          FOLLOW_UP_2: 'FU #2',
          SET_MEETING: 'Set MTG',
          FOLLOW_UP_MEETING_1: 'Confirm #1',
          FOLLOW_UP_MEETING_2: 'Confirm #2',
          PREPARE_MEETING: 'Prep MTG',
          WAIT: 'Wait',
          FOLLOW_UP: 'FU Deal',
          DROP: 'Drop',
        };
        return (
          <Badge
            variant="secondary"
            className={stepColors[step] || 'bg-gray-100 text-gray-700'}
          >
            {stepLabels[step] || step}
          </Badge>
        );
      },
    },
    {
      accessorKey: 'city',
      sortable: true,
      header: 'City',
    },
    {
      accessorKey: 'country',
      sortable: true,
      header: 'Country',
    },
    {
      accessorKey: 'action',
      header: 'Action',
      cell: ({ row }) => (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <MainButton icon={MoreVertical} size="icon" variant="ghost" />
          </DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuItem
              onClick={() => {
                setSelectedRow(row);
                setIsDialogOpen(true);
              }}
            >
              <Edit2 /> Edit
            </DropdownMenuItem>
            <DropdownMenuItem
              className="text-destructive focus:bg-destructive-secondary focus:text-destructive"
              onClick={() => handleDelete([row.id])}
            >
              <Trash2 /> Delete
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      ),
    },
  ];

  const rowSelectorExtraContent = (
    <>
      <CreateListContactDialog
        onSuccess={() => setSelectedId({})}
        contactIds={selectedId[pageKey] || []}
        filterValue={filter}
      >
        <AddButton variant="secondary" text="Create list" />
      </CreateListContactDialog>
      <DeleteButton
        text="Delete"
        onConfirm={() => handleDelete(selectedId[pageKey] || [])}
        description="This action cannot be undone. This will permanently delete your contacts and remove
your data from our servers."
      />
    </>
  );

  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [isDialogOpenHubSpot, setIsDialogOpenHubSpot] = useState(false);

  const segmentsList = segments?.data || [];
  const selectedSegment =
    selectedSegmentId !== '__all__'
      ? segmentsList.find((s) => s.id === selectedSegmentId)
      : undefined;

  const headerExtraContent = (
    <>
      <div className="flex items-center gap-2">
        <Filter className="h-4 w-4 text-muted-foreground" />
        <Select
          value={selectedStatusFilter}
          onValueChange={(value) => {
            setSelectedStatusFilter(value);
            if (value && value !== '__all__') {
              setFilter({ ...filter, status: value });
              return;
            }
            const { status, ...rest } = filter;
            setFilter(rest);
          }}
        >
          <SelectTrigger className="w-[140px]">
            <SelectValue placeholder="All Status" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="__all__">All Status</SelectItem>
            <SelectItem value="ready">Ready</SelectItem>
            <SelectItem value="pending">Pending</SelectItem>
          </SelectContent>
        </Select>
        <Select
          value={selectedSegmentId}
          onValueChange={(value) => {
            setSelectedSegmentId(value);
            if (value && value !== '__all__') {
              setFilter({ ...filter, segment_id: value });
              return;
            }
            const { segment_id, ...rest } = filter;
            setFilter(rest);
          }}
        >
          <SelectTrigger className="w-[200px]">
            <SelectValue placeholder="All Segments" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="__all__">All Segments</SelectItem>
            {segmentsList.map((segment) => (
              <SelectItem key={segment.id} value={segment.id}>
                {segment.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <MainButton
        icon={Star}
        text={priorityFilter ? 'Show All' : 'Priority Contacts'}
        variant={priorityFilter ? 'default' : 'secondary'}
        onClick={() => setPriorityFilter(!priorityFilter)}
        id="contacts-priority-button"
      />
      <AddButton
        variant="secondary"
        text="Create"
        onClick={() => setIsDialogOpen(true)}
        id="contacts-create-button"
      />
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <MainButton
            icon={CloudUpload}
            text="Import"
            id="contacts-import-button"
          />
        </DropdownMenuTrigger>
        <DropdownMenuContent>
          <DropdownMenuLabel>Import</DropdownMenuLabel>
          <DropdownMenuItem onClick={() => setIsDialogOpenHubSpot(true)}>
            <IconHubspot /> From HubSpot
          </DropdownMenuItem>
          <DropdownMenuItem
            onClick={() => router.push('/all-prospects/csv-import')}
          >
            <IconCSV /> From CSV
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuLabel>Maintenance</DropdownMenuLabel>
          <DropdownMenuItem
            onClick={() => recalculateStatusMutation.mutate()}
            disabled={recalculateStatusMutation.isPending}
          >
            <RefreshCw
              className={
                recalculateStatusMutation.isPending ? 'animate-spin' : ''
              }
            />
            {recalculateStatusMutation.isPending
              ? 'Recalculating...'
              : 'Recalculate Status'}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </>
  );

  const { start, reset } = useTour([
    {
      element: '#contacts-import-button',
      popover: {
        title: '',
        side: 'left',
        description: '',
        customContent: createTourCustomContent({
          title: 'Import',
          description: (
            <DescriptionText
              title="Add Contacts"
              description="Bring in leads by uploading a file or syncing with your CRM."
            />
          ),
          imageSrc: '',
          user,
          onNext: async (step: number) => {
            await UpdateOnboarding(
              {
                contacts: {
                  onboarding: true,
                  step: step,
                  status: 'in_progress',
                },
              },
              user
            );
          },
          onPrev: (step: number) => {
            UpdateOnboarding(
              {
                contacts: {
                  onboarding: true,
                  step: step,
                  status: 'in_progress',
                },
              },
              user
            );
          },
          onFinish: async (step: number) => {
            await UpdateOnboarding(
              {
                contacts: {
                  onboarding: false,
                  step: step,
                  status: 'completed',
                },
              },
              user
            );
            await invalidate();
          },
          onSkipAll: async (step: number) => {
            await UpdateOnboarding(
              {
                contacts: {
                  onboarding: false,
                  step: step,
                  status: 'skipped',
                },
              },
              user
            );
            await invalidate();
          },
          onNavigate: (step: number) => {
            UpdateOnboarding(
              {
                campaigns: {
                  onboarding: true,
                  step: step,
                  status: 'in_progress',
                },
              },
              user
            );
          },
        }),
      },
    },
    {
      element: '#contacts-create-button',
      popover: {
        title: '',
        side: 'left',
        description: '',
        customContent: createTourCustomContent({
          title: 'Create',
          description: (
            <DescriptionText
              title="Create Contact Manually"
              description="Or enter a lead’s info manually to start personal outreach."
            />
          ),
          imageSrc: '',
          user,
          onNext: (step: number) => {
            UpdateOnboarding(
              {
                contacts: {
                  onboarding: true,
                  step: step,
                  status: 'in_progress',
                },
              },
              user
            );
          },
          onPrev: (step: number) => {
            UpdateOnboarding(
              {
                contacts: {
                  onboarding: true,
                  step: step,
                  status: 'in_progress',
                },
              },
              user
            );
          },
          onFinish: async (step: number) => {
            await UpdateOnboarding(
              {
                contacts: {
                  onboarding: false,
                  step: step,
                  status: 'completed',
                },
              },
              user
            );
            await invalidate();
          },
          onSkipAll: async (step: number) => {
            await UpdateOnboarding(
              {
                contacts: {
                  onboarding: false,
                  step: step,
                  status: 'skipped',
                },
              },
              user
            );
            await invalidate();
          },
          onNavigate: (step: number) => {
            UpdateOnboarding(
              {
                campaigns: {
                  onboarding: true,
                  step: step,
                  status: 'in_progress',
                },
              },
              user
            );
          },
        }),
      },
    },
    {
      element: '#contacts-table-filter-button',
      popover: {
        title: '',
        side: 'right',
        description: '',
        customContent: createTourCustomContent({
          title: 'Filters',
          description: (
            <DescriptionText
              title="Find Fast with Filters"
              description="Segment contacts by tag, stage, or source to manage outreach better."
            />
          ),
          imageSrc: '',
          user,
          onNext: (step: number) => {
            UpdateOnboarding(
              {
                contacts: {
                  onboarding: true,
                  step: step,
                  status: 'in_progress',
                },
              },
              user
            );
          },
          onPrev: (step: number) => {
            UpdateOnboarding(
              {
                contacts: {
                  onboarding: true,
                  step: step,
                  status: 'in_progress',
                },
              },
              user
            );
          },
          onFinish: async (step: number) => {
            await UpdateOnboarding(
              {
                contacts: {
                  onboarding: false,
                  step: step,
                  status: 'completed',
                },
              },
              user
            );
            await invalidate();
          },
          onSkipAll: async (step: number) => {
            await UpdateOnboarding(
              {
                contacts: {
                  onboarding: false,
                  step: step,
                  status: 'skipped',
                },
              },
              user
            );
            await invalidate();
          },
          onNavigate: (step: number) => {
            UpdateOnboarding(
              {
                campaigns: {
                  onboarding: true,
                  step: step,
                  status: 'in_progress',
                },
              },
              user
            );
          },
        }),
      },
    },
  ]);

  useEffect(() => {
    if (
      user?.ui_metadata?.contacts?.onboarding &&
      contacts?.data?.list.length === 0
    ) {
      start();
    }
    return () => {
      reset();
    };
  }, [user, contacts]);

  return (
    <ContentLayout
      title="Contacts"
      section="Contacts"
      icon={Users}
      className="md:w-[calc(100vw-260px)]"
    >
      <PageHero
        icon={Users}
        eyebrow="Contacts"
        accent="rose"
        title="Contacts"
        description="Your prospect database. Each contact carries AI-enriched insights, outreach state, and segment scores — feeding every campaign downstream."
        actions={
          <div className="rounded-xl border bg-background/70 px-6 py-4 text-center">
            <p className="text-3xl font-bold leading-none">{pageCount}</p>
            <p className="mt-1 text-[0.8rem] text-muted-foreground">Contacts</p>
          </div>
        }
      />
      {selectedSegmentId !== '__all__' && selectedSegment && (
        <div className="mb-4 flex items-center gap-2 rounded-lg border border-blue-200 bg-blue-50 p-3 dark:border-blue-800 dark:bg-blue-950">
          <Filter className="h-4 w-4 text-blue-600 dark:text-blue-400" />
          <span className="text-sm">
            Filtering by segment:{' '}
            <span className="font-semibold">{selectedSegment.name}</span>
          </span>
          <button
            onClick={() => {
              setSelectedSegmentId('__all__');
              const { segment_id, ...rest } = filter;
              setFilter(rest);
            }}
            className="ml-auto text-sm text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-200"
          >
            Clear filter
          </button>
        </div>
      )}
      {error ? (
        <p className="p-2 text-center text-destructive">
          Error loading contacts
        </p>
      ) : (
        <DataTable
          data={data}
          loading={isLoading}
          columns={
            isAdmin
              ? columns
              : columns.filter((col) => col.accessorKey !== 'added_by_name')
          }
          pageCount={pageCount}
          pageSize={pageSize}
          rowSelectorExtraContent={rowSelectorExtraContent}
          pageKey={pageKey}
          filterIdButton="contacts-table-filter-button"
          headerExtraContent={headerExtraContent}
          onFilterChange={setFilter}
          filterFieldSelections={_fieldsValue}
          renderExpandedRow={(contact) => (
            <ContactExpandedRow contact={contact} />
          )}
          getExpandableIndicator={(contact) => {
            const hasCustomFields =
              Object.keys(contact.profile?.custom_fields || {}).length > 0;
            const hasResearchFindings =
              (contact.profile?.research_findings || []).length > 0;
            const hasAIInsights =
              (contact.ai_insights?.suspected_pain_points?.length ?? 0) > 0 ||
              (contact.ai_insights?.suspected_goals?.length ?? 0) > 0 ||
              (contact.ai_insights?.buying_signals?.length ?? 0) > 0;
            // Allow expansion even if empty to show “no data” state
            return (
              hasCustomFields || hasResearchFindings || hasAIInsights || true
            );
          }}
        />
      )}
      <CreateContactDialog
        // The form reads custom fields once, when it mounts. Opened before
        // they loaded, it filed their values under "Additional fields" and
        // saved the old values over the user's edits; remount once they load.
        key={customFields ? 'fields-loaded' : 'fields-loading'}
        open={isDialogOpen}
        onOpenChange={(open) => {
          setIsDialogOpen(open);
          if (!open) {
            setSelectedRow(null);
          }
        }}
        onSuccess={() => {
          createSuccess();
          setIsDialogOpen(false);
        }}
        data={selectedRow}
        isEdit={!!selectedRow}
        customFields={customFields?.data?.list || []}
      />
      <ImportFromHubSpotDialog
        onSuccess={() => {
          refetchContacts();
          refetchFieldsValue();
          setIsDialogOpenHubSpot(false);
        }}
        open={isDialogOpenHubSpot}
        onOpenChange={setIsDialogOpenHubSpot}
      />
    </ContentLayout>
  );
}

interface CreateListContactDialogProps extends PropsWithChildren {
  onSuccess: () => void;
  contactIds: string[];
  filterValue?: Record<string, string>;
}

function CreateListContactDialog({
  children,
  onSuccess,
  contactIds,
  filterValue,
}: CreateListContactDialogProps) {
  const [isDialogOpen, setIsDialogOpen] = useState(false);

  const { mutate, isPending } = useMutation({
    mutationFn: contactListApi.create,
    onSuccess: () => {
      onSuccess();
      toast.success('Create contact list successfully');
    },
    onError: () => {
      toast.error('Create contact list failed');
    },
  });

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    const formData = new FormData(e.target as HTMLFormElement);
    mutate({ name: formData.get('name') as string, contact_ids: contactIds });
  };

  return (
    <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
      <DialogTrigger asChild>{children}</DialogTrigger>
      <DialogContent className="sm:max-w-[640px]">
        <DialogHeader>
          <DialogTitle>Add a New Contact List</DialogTitle>
          <DialogDescription>
            Organize your contacts by creating a new list. Add a name,
            description, and categorize your contacts for better management and
            targeted communication.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-6">
          <div className="space-y-2">
            <Label htmlFor="input-01">List name</Label>
            <Input
              id="input-01"
              name="name"
              placeholder="Enter list name"
              type="text"
              required
            />
          </div>
          <div className="space-y-2">
            <p>Selected contacts</p>
            <div className="rounded-md bg-gray-100 p-4">
              <p>{` ${contactIds.length} contacts selected `}</p>
              {(filterValue?.job_title || filterValue?.company) && (
                <p>Active Filters:</p>
              )}
              {filterValue?.job_title && (
                <p>
                  <span className="font-semibold">Job title:</span>{' '}
                  {filterValue.job_title}
                </p>
              )}
              {filterValue?.company && (
                <p>
                  <span className="font-semibold">Company:</span>{' '}
                  {filterValue.company}
                </p>
              )}
            </div>
          </div>
          <MainButton
            text="Create"
            type="submit"
            className="w-full"
            loading={isPending}
          />
        </form>
      </DialogContent>
    </Dialog>
  );
}
