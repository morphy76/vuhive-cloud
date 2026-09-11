import React from 'react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { AlertTriangle, Loader2 } from 'lucide-react'
import { useDeleteProfile } from '@/hooks/use-profiles'
import { useToast } from '@/hooks/use-toast'
import type { RunnerProfile } from '@/types/profile'

interface DeleteProfileDialogProps {
  profile: RunnerProfile | null
  open: boolean
  onOpenChange: (open: boolean) => void
}

export const DeleteProfileDialog: React.FC<DeleteProfileDialogProps> = ({
  profile,
  open,
  onOpenChange,
}) => {
  const deleteMutation = useDeleteProfile()
  const { toast } = useToast()

  if (!profile) return null

  const handleDelete = async () => {
    try {
      await deleteMutation.mutateAsync(profile.id)
      toast({
        title: 'Profile Deleted',
        description: `Runner profile "${profile.name}" has been deleted.`,
      })
      onOpenChange(false)
    } catch (err: any) {
      toast({
        variant: 'destructive',
        title: 'Deletion Failed',
        description: err.message || 'Failed to delete runner profile.',
      })
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-full bg-rose-100 text-rose-600 dark:bg-rose-950/60 dark:text-rose-400">
              <AlertTriangle className="h-5 w-5" />
            </div>
            <div>
              <DialogTitle>Delete Runner Profile</DialogTitle>
              <DialogDescription className="mt-1">
                Are you sure you want to delete profile{' '}
                <span className="font-semibold text-slate-900 dark:text-slate-100">
                  {profile.name}
                </span>
                ?
              </DialogDescription>
            </div>
          </div>
        </DialogHeader>

        <p className="text-xs text-slate-500 dark:text-slate-400">
          This action will immediately remove the runner configuration from the control plane registry.
          Existing completed runs referencing this profile will retain their historical record.
        </p>

        <DialogFooter className="gap-2 sm:gap-0 mt-4">
          <Button
            type="button"
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={deleteMutation.isPending}
          >
            Cancel
          </Button>
          <Button
            type="button"
            variant="destructive"
            onClick={handleDelete}
            disabled={deleteMutation.isPending}
            className="gap-2"
          >
            {deleteMutation.isPending && <Loader2 className="w-4 h-4 animate-spin" />}
            <span>Delete Profile</span>
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
