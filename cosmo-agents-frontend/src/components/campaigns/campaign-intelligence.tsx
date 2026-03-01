'use client';

import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { BarChart3, Lightbulb } from 'lucide-react';

export function CampaignIntelligence() {
  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <BarChart3 className="h-5 w-5" />
            Campaign Intelligence
          </CardTitle>
          <CardDescription>
            Connect your campaign data to see performance metrics, segment analysis, and AI-powered recommendations
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col items-center justify-center py-12 text-center">
            <Lightbulb className="h-16 w-16 text-muted-foreground mb-4" />
            <h3 className="text-lg font-semibold mb-2">No Intelligence Data Available</h3>
            <p className="text-sm text-muted-foreground max-w-md">
              Campaign intelligence features require real campaign data from your email service provider or CRM integration.
              Once connected, you'll see metrics, segment performance, and AI recommendations here.
            </p>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
