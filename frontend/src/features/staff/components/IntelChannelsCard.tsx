import { useEffect, useState } from "react";
import { Check, Pencil, Trash2, X } from "lucide-react";
import { pb } from "@/config/pb";

export default function IntelChannelsCard() {
  const [channels, setChannels] = useState<
    { id: string; channel_name: string }[]
  >([]);
  const [newChannel, setNewChannel] = useState("");
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editingName, setEditingName] = useState("");

  const loadChannels = () => {
    pb.collection("intel_channels")
      .getFullList({ sort: "channel_name" })
      .then((records) => {
        setChannels(
          records.map((record) => ({
            id: record.id,
            channel_name: record.channel_name as string,
          })),
        );
      })
      .catch(() => setChannels([]));
  };

  useEffect(() => {
    loadChannels();
  }, []);

  const addChannel = async () => {
    if (!newChannel) return;
    await pb.collection("intel_channels").create({ channel_name: newChannel });
    setNewChannel("");
    loadChannels();
  };

  const deleteChannel = async (id: string) => {
    await pb.collection("intel_channels").delete(id);
    loadChannels();
  };

  const startEdit = (id: string, name: string) => {
    setEditingId(id);
    setEditingName(name);
  };

  const cancelEdit = () => {
    setEditingId(null);
    setEditingName("");
  };

  const saveEdit = async () => {
    if (!editingId || !editingName.trim()) return;
    await pb
      .collection("intel_channels")
      .update(editingId, { channel_name: editingName.trim() });
    setEditingId(null);
    setEditingName("");
    loadChannels();
  };

  return (
    <section className="card bg-base-200/70 border border-slate-800">
      <div className="card-body space-y-4">
        <div>
          <h2 className="font-display text-2xl">Intel Channels</h2>
          <p className="text-sm text-slate-400">
            Manage the channels used by the uploader.
          </p>
        </div>
        <div className="flex gap-2">
          <input
            className="input input-sm input-bordered bg-base-300"
            value={newChannel}
            onChange={(e) => setNewChannel(e.target.value)}
            placeholder="Channel name"
          />
          <button
            className="btn btn-sm btn-info btn-outline"
            onClick={addChannel}
          >
            Add
          </button>
        </div>
        <ul className="space-y-2 text-sm">
          {channels.length === 0 && (
            <li className="text-slate-500">No channels configured.</li>
          )}
          {channels.map((channel) => {
            const isEditing = editingId === channel.id;
            return (
              <li
                key={channel.id}
                className="flex items-center justify-between rounded-lg border border-slate-800/70 bg-base-300/40 px-3 py-2"
              >
                {isEditing ? (
                  <input
                    className="input input-xs input-bordered bg-base-300 flex-1 mr-3"
                    value={editingName}
                    onChange={(e) => setEditingName(e.target.value)}
                    autoFocus
                  />
                ) : (
                  <span className="font-medium text-slate-100">
                    {channel.channel_name}
                  </span>
                )}
                <div className="flex items-center gap-1">
                  {isEditing ? (
                    <>
                      <button
                        className="btn btn-xs btn-outline btn-square"
                        onClick={saveEdit}
                        aria-label="Save channel"
                      >
                        <Check className="h-4 w-4" />
                      </button>
                      <button
                        className="btn btn-xs btn-outline btn-square"
                        onClick={cancelEdit}
                        aria-label="Cancel edit"
                      >
                        <X className="h-4 w-4" />
                      </button>
                    </>
                  ) : (
                    <>
                      <button
                        className="btn btn-xs btn-outline btn-square"
                        onClick={() =>
                          startEdit(channel.id, channel.channel_name)
                        }
                        aria-label="Edit channel"
                      >
                        <Pencil className="h-4 w-4" />
                      </button>
                      <button
                        className="btn btn-xs btn-outline btn-square btn-error"
                        onClick={() => deleteChannel(channel.id)}
                        aria-label="Delete channel"
                      >
                        <Trash2 className="h-4 w-4" />
                      </button>
                    </>
                  )}
                </div>
              </li>
            );
          })}
        </ul>
      </div>
    </section>
  );
}
