import React, { useState } from 'react';
import { Card, CardContent, CardHeader } from '../components/atoms/Card';
import { Button } from '../components/atoms/Button';
import { Input } from '../components/atoms/Input';
import { Select } from '../components/atoms/Select';
import { Switch } from '../components/atoms/Switch';

interface AISettings {
  enabled: boolean;
  provider: string;
  apiKey: string;
  model: string;
  batchSize: number;
  confidenceThreshold: number;
}

const AISettings: React.FC = () => {
  const [settings, setSettings] = useState<AISettings>({
    enabled: false,
    provider: 'openai',
    apiKey: '',
    model: 'gpt-4',
    batchSize: 10,
    confidenceThreshold: 0.85,
  });

  const handleSave = async () => {
    try {
      await fetch('/api/v1/settings/ai', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(settings),
      });
      // Show success message
    } catch (error) {
      // Show error message
      console.error('Failed to save AI settings:', error);
    }
  };

  return (
    <div className="container mx-auto px-4 py-8">
      <h1 className="text-2xl font-bold mb-6">AI Settings</h1>
      
      <Card>
        <CardHeader>
          <h2 className="text-xl">AI Service Configuration</h2>
        </CardHeader>
        <CardContent>
          <div className="space-y-6">
            {/* Enable/Disable AI */}
            <div className="flex items-center justify-between">
              <label className="font-medium">Enable AI Service</label>
              <Switch
                checked={settings.enabled}
                onChange={(checked) => setSettings({ ...settings, enabled: checked })}
              />
            </div>

            {/* Provider Selection */}
            <div>
              <label className="block font-medium mb-2">AI Provider</label>
              <Select
                value={settings.provider}
                onChange={(e) => setSettings({ ...settings, provider: e.target.value })}
                className="w-full"
              >
                <option value="openai">OpenAI</option>
                <option value="anthropic">Anthropic</option>
                <option value="local">Local Model</option>
              </Select>
            </div>

            {/* API Key */}
            <div>
              <label className="block font-medium mb-2">API Key</label>
              <Input
                type="password"
                value={settings.apiKey}
                onChange={(e) => setSettings({ ...settings, apiKey: e.target.value })}
                className="w-full"
                placeholder="Enter your API key"
              />
            </div>

            {/* Model Selection */}
            <div>
              <label className="block font-medium mb-2">Model</label>
              <Select
                value={settings.model}
                onChange={(e) => setSettings({ ...settings, model: e.target.value })}
                className="w-full"
              >
                <option value="gpt-4">GPT-4</option>
                <option value="gpt-3.5-turbo">GPT-3.5 Turbo</option>
                <option value="claude-3">Claude 3</option>
              </Select>
            </div>

            {/* Batch Size */}
            <div>
              <label className="block font-medium mb-2">Batch Size</label>
              <Input
                type="number"
                value={settings.batchSize}
                onChange={(e) => setSettings({ ...settings, batchSize: parseInt(e.target.value) })}
                className="w-full"
                min={1}
                max={100}
              />
            </div>

            {/* Confidence Threshold */}
            <div>
              <label className="block font-medium mb-2">Confidence Threshold</label>
              <Input
                type="number"
                value={settings.confidenceThreshold}
                onChange={(e) => setSettings({ ...settings, confidenceThreshold: parseFloat(e.target.value) })}
                className="w-full"
                min={0}
                max={1}
                step={0.01}
              />
            </div>

            {/* Save Button */}
            <div className="flex justify-end">
              <Button onClick={handleSave} variant="primary">
                Save Settings
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
};

export default AISettings; 