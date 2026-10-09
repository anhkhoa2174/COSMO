'use client';

import { AddButton } from '@/components/buttons/add-button';
import { DeleteButton } from '@/components/buttons/delete-button';
import { Button } from '@/components/ui/button';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
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
import PersonalApiKeysApi, {
  PersonalApiKey,
} from '@/network/client/personal-apikeys';
import { useUser } from '@/hooks/use-user';
import { Calendar, CheckCircle, Copy, Key, Shield } from 'lucide-react';
import { useEffect, useState } from 'react';
import { toast } from 'sonner';

export default function ApiKeysPage() {
  const { user } = useUser();
  const [apiKeys, setApiKeys] = useState<PersonalApiKey[]>([]);
  const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false);
  const [newKeyName, setNewKeyName] = useState('');
  const [expirationDays, setExpirationDays] = useState('30'); // Default to 30 days
  const [isLoading, setIsLoading] = useState(false);
  const [isLoadingKeys, setIsLoadingKeys] = useState(true);
  const [visibleKeys, setVisibleKeys] = useState<Set<string>>(new Set());
  const [showRawKeyDialog, setShowRawKeyDialog] = useState(false);
  const [newRawKey, setNewRawKey] = useState('');
  // Load API keys from server
  useEffect(() => {
    const loadApiKeys = async () => {
      if (!user?.id) return;

      try {
        setIsLoadingKeys(true);
        const response = await PersonalApiKeysApi.list({ userId: user.id });
        setApiKeys(response.data.list || []);
      } catch (error) {
        console.error('Failed to load API keys:', error);
        toast.error('Failed to load API keys');
      } finally {
        setIsLoadingKeys(false);
      }
    };

    loadApiKeys();
  }, [user?.id]);

  const handleCreateApiKey = async () => {
    if (!newKeyName.trim()) {
      toast.error('Please enter a name for your API key');
      return;
    }

    if (!user?.id) {
      toast.error('User not authenticated');
      return;
    }

    setIsLoading(true);
    try {
      // Calculate expiration timestamp based on selected days
      const expirationMs = parseInt(expirationDays) * 24 * 60 * 60 * 1000;
      const expiresAt = Math.floor((Date.now() + expirationMs) / 1000); // Convert to Unix timestamp

      const response = await PersonalApiKeysApi.create({
        userId: user.id,
        body: {
          name: newKeyName,
          expires_at: expiresAt,
        },
      });
      if (response) {
        // Store the raw key and show the dialog
        setNewRawKey(response?.data?.raw_key || '');
        setShowRawKeyDialog(true);

        // Reload the API keys list
        const updatedResponse = await PersonalApiKeysApi.list({
          userId: user.id,
        });
        setApiKeys(updatedResponse.data.list || []);

        setNewKeyName('');
        setExpirationDays('30'); // Reset to default
        setIsCreateDialogOpen(false);
      }
    } catch (error) {
      console.error('Failed to create API key:', error);
      toast.error('Failed to create API key');
    } finally {
      setIsLoading(false);
    }
  };

  const handleDeleteApiKey = async (keyId: string) => {
    if (!user?.id) {
      toast.error('User not authenticated');
      return;
    }

    try {
      await PersonalApiKeysApi.delete({
        userId: user.id,
        personalApiKey: keyId,
      });

      setApiKeys((prev) => prev.filter((key: any) => key?.id !== keyId));
      toast.success('API key deleted successfully!');
    } catch (error) {
      console.error('Failed to delete API key:', error);
      toast.error('Failed to delete API key');
    }
  };

  const toggleKeyVisibility = (keyId: string) => {
    setVisibleKeys((prev) => {
      const newSet = new Set(prev);
      if (newSet.has(keyId)) {
        newSet.delete(keyId);
      } else {
        newSet.add(keyId);
      }
      return newSet;
    });
  };

  const copyToClipboard = async (text: string) => {
    try {
      await navigator.clipboard.writeText(text);
      toast.success('API key copied to clipboard!');
    } catch (error) {
      toast.error('Failed to copy to clipboard');
    }
  };

  const handleCloseRawKeyDialog = () => {
    setShowRawKeyDialog(false);
    setNewRawKey('');
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  if (!user?.id) {
    return (
      <div className="flex items-center justify-center py-12">
        <p className="text-muted-foreground">
          Please sign in to manage your API keys.
        </p>
      </div>
    );
  }

  const maskApiKey = (key: string) => {
    return `${key?.substring(0, 7)}${'•'.repeat(20)}${key?.substring(key.length - 4)}`;
  };

  return (
    <div className="space-y-6 p-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">API Keys</h1>
          <p className="text-muted-foreground">
            Manage your personal API keys for accessing the platform
          </p>
        </div>
        <Dialog open={isCreateDialogOpen} onOpenChange={setIsCreateDialogOpen}>
          <DialogTrigger asChild>
            <AddButton text="Create API Key" />
          </DialogTrigger>
          <DialogContent className="sm:max-w-[425px]">
            <DialogHeader>
              <DialogTitle className="flex items-center gap-2">
                <Key className="h-5 w-5" />
                Create New API Key
              </DialogTitle>
              <DialogDescription>
                Create a new API key to access your account programmatically.
              </DialogDescription>
            </DialogHeader>
            <div className="grid gap-4 py-4">
              <div className="grid gap-2">
                <Label htmlFor="name">Name</Label>
                <Input
                  id="name"
                  placeholder="e.g., Production API Key"
                  value={newKeyName}
                  onChange={(e) => setNewKeyName(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      handleCreateApiKey();
                    }
                  }}
                />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="expiration">Expiration</Label>
                <Select
                  value={expirationDays}
                  onValueChange={setExpirationDays}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="Select expiration time" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="7">7 days</SelectItem>
                    <SelectItem value="30">30 days</SelectItem>
                    <SelectItem value="60">60 days</SelectItem>
                    <SelectItem value="90">90 days</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
            <DialogFooter>
              <Button
                variant="outline"
                onClick={() => setIsCreateDialogOpen(false)}
              >
                Cancel
              </Button>
              <Button onClick={handleCreateApiKey} disabled={isLoading}>
                {isLoading ? 'Creating...' : 'Create API Key'}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>
      {/* Usage Guidelines */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Usage Guidelines</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-4 md:grid-cols-2">
            <div className="space-y-2">
              <h4 className="flex items-center gap-2 font-medium">
                <Shield className="h-4 w-4 text-green-600" />
                Security & Storage
              </h4>
              <ul className="space-y-1 text-sm text-muted-foreground">
                <li>
                  • Copy your API key immediately after creation - it's only
                  shown once
                </li>
                <li>
                  • Store keys securely using environment variables or secure
                  vaults
                </li>
                <li>
                  • Never commit API keys to version control or share publicly
                </li>
                <li>
                  • Set appropriate expiration times (7-90 days) for better
                  security
                </li>
                <li>
                  • Delete expired or unused keys to minimize security risks
                </li>
              </ul>
            </div>
            <div className="space-y-2">
              <h4 className="flex items-center gap-2 font-medium">
                <Key className="h-4 w-4 text-blue-600" />
                Integration Guide
              </h4>
              <ul className="space-y-1 text-sm text-muted-foreground">
                <li>• Add your API key to the Authorization header</li>
                <li>
                  • Format:{' '}
                  <code className="rounded bg-muted px-1">
                    Authorization: Bearer your-api-key
                  </code>
                </li>
                <li>
                  • Each key has a specific expiration date - monitor usage
                </li>
                <li>
                  • Create separate keys for different environments (dev/prod)
                </li>
                <li>• Refer to our API documentation for endpoint details</li>
              </ul>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Raw Key Display Dialog */}
      <Dialog open={showRawKeyDialog} onOpenChange={setShowRawKeyDialog}>
        <DialogContent className="sm:max-w-[500px]">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <CheckCircle className="h-5 w-5 text-green-600" />
              API Key Created Successfully!
            </DialogTitle>
            <DialogDescription>
              Your API key has been created. Please copy it now as you won't be
              able to see it again.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label className="text-sm font-medium">Your API Key</Label>
              <div className="flex items-center gap-2">
                <div className="flex-1 break-all rounded-md border bg-muted/50 px-3 py-2 font-mono text-sm">
                  {newRawKey}
                </div>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => copyToClipboard(newRawKey)}
                >
                  <Copy className="h-4 w-4" />
                </Button>
              </div>
            </div>
            <div className="rounded-md border border-yellow-200 bg-yellow-50 p-3">
              <div className="flex items-start gap-2">
                <Shield className="mt-0.5 h-4 w-4 text-yellow-600" />
                <div className="text-sm text-yellow-800">
                  <strong>Important:</strong> This is the only time you'll see
                  this key. Make sure to copy and store it securely.
                </div>
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button onClick={handleCloseRawKeyDialog} className="w-full">
              I've copied my key
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* API Keys List */}
      <div className="space-y-4">
        {isLoadingKeys ? (
          <Card>
            <CardContent className="flex items-center justify-center py-12">
              <div className="flex items-center gap-2">
                <div className="h-4 w-4 animate-spin rounded-full border-2 border-primary border-t-transparent" />
                <span className="text-muted-foreground">
                  Loading API keys...
                </span>
              </div>
            </CardContent>
          </Card>
        ) : apiKeys.length === 0 ? (
          <Card>
            <CardContent className="flex flex-col items-center justify-center py-12">
              <Key className="mb-4 h-12 w-12 text-muted-foreground" />
              <h3 className="mb-2 text-lg font-semibold">No API keys yet</h3>
              <p className="mb-4 text-center text-muted-foreground">
                Create your first API key to start using the platform
                programmatically.
              </p>
              <AddButton
                text="Create Your First API Key"
                onClick={() => setIsCreateDialogOpen(true)}
              />
            </CardContent>
          </Card>
        ) : (
          apiKeys.length > 0 &&
          apiKeys.map((apiKey: any) => (
            <Card key={apiKey?.id} className="transition-all hover:shadow-md">
              <CardHeader className="pb-3">
                <div className="flex items-start justify-between">
                  <div className="space-y-1">
                    <CardTitle className="flex items-center gap-2">
                      <Shield className="h-4 w-4" />
                      {apiKey?.name}
                    </CardTitle>
                    <CardDescription className="flex items-center gap-4">
                      <span className="flex items-center gap-1">
                        <Calendar className="h-3 w-3" />
                        Created {formatDate(apiKey?.created_at)}
                      </span>
                      {apiKey?.last_used_at && (
                        <span className="text-xs">
                          Last used {formatDate(apiKey?.last_used_at)}
                        </span>
                      )}
                    </CardDescription>
                  </div>
                  <DeleteButton
                    onConfirm={() => handleDeleteApiKey(apiKey?.id)}
                    description={`This will permanently delete the API key "${apiKey?.name}". This action cannot be undone.`}
                    size="sm"
                    variant="destructive-secondary"
                  />
                </div>
              </CardHeader>
              <CardContent className="space-y-4">
                {/* API Key Display */}
                <div className="space-y-2">
                  <Label className="text-xs font-medium text-muted-foreground">
                    API KEY
                  </Label>
                  <div className="flex items-center gap-2">
                    <div className="flex-1 rounded-md border bg-muted/50 px-3 py-2 font-mono text-sm">
                      {maskApiKey(apiKey?.prefix || '')}
                    </div>
                  </div>
                </div>
              </CardContent>
            </Card>
          ))
        )}
      </div>
    </div>
  );
}
