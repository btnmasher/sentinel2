import { useEffect, useState } from "react";
import { Check, Pencil, Plus, Trash2, X } from "lucide-react";
import { pb } from "@/config/pb";
import Panel from "@/components/Panel";
import ShadowedScrollArea from "@/components/ShadowedScrollArea";

export default function IntelChannelsCard() {
  const [channels, setChannels] = useState<
    { id: string; channel_name: string }[]
  >([]);
  const [isAdding, setIsAdding] = useState(false);
  const [newChannelName, setNewChannelName] = useState("");
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editingName, setEditingName] = useState("");
  const [listHasOverflow, setListHasOverflow] = useState(false);

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
    if (!newChannelName.trim()) return;
    await pb
      .collection("intel_channels")
      .create({ channel_name: newChannelName.trim() });
    setNewChannelName("");
    setIsAdding(false);
    loadChannels();
  };

  const deleteChannel = async (id: string) => {
    await pb.collection("intel_channels").delete(id);
    loadChannels();
  };

  const startEdit = (id: string, name: string) => {
    setIsAdding(false);
    setNewChannelName("");
    setEditingId(id);
    setEditingName(name);
  };

  const cancelEdit = () => {
    setEditingId(null);
    setEditingName("");
  };

  const startAdd = () => {
    cancelEdit();
    setIsAdding(true);
  };

  const cancelAdd = () => {
    setIsAdding(false);
    setNewChannelName("");
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

  const addSection = isAdding ? (
    <div className="rounded-lg border border-dashed border-slate-700/80 bg-base-300/20 p-2">
      <div className="flex items-center justify-between gap-2">
        <input
          className="input input-xs input-bordered bg-base-300 flex-1"
          value={newChannelName}
          onChange={(e) => setNewChannelName(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") {
              e.preventDefault();
              void addChannel();
            }
          }}
          placeholder="Channel name"
          autoFocus
        />
        <div className="flex items-center gap-1">
          <button
            className="btn btn-xs btn-outline btn-square"
            onClick={addChannel}
            aria-label="Save new channel"
          >
            <Check className="h-4 w-4" />
          </button>
          <button
            className="btn btn-xs btn-outline btn-square"
            onClick={cancelAdd}
            aria-label="Cancel add"
          >
            <X className="h-4 w-4" />
          </button>
        </div>
      </div>
    </div>
  ) : (
    <button
      className="w-full rounded-lg border border-dashed border-slate-700/80 bg-base-300/20 px-3 py-2 text-slate-300 transition hover:bg-base-300/40 hover:text-slate-100"
      onClick={startAdd}
      aria-label="Add channel"
    >
      <span className="flex items-center justify-center gap-2 text-sm">
        <Plus className="h-4 w-4" />
        Add channel
      </span>
    </button>
  );

  return (
    <Panel
      title="Intel Channels"
      hint="Manage the channels used by the uploader."
      className="h-full min-h-0"
      bodyClassName="flex h-full min-h-0 flex-col gap-4 overflow-hidden"
    >
      <div className="flex min-h-0 flex-1 flex-col gap-2 text-sm">
        <ShadowedScrollArea
          className="min-h-0 flex-1"
          scrollClassName="pr-1"
          onStateChange={(state) => setListHasOverflow(state.hasOverflow)}
        >
          <div className="space-y-2">
            <ul className="space-y-2">
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
            {!listHasOverflow ? addSection : null}
          </div>
        </ShadowedScrollArea>

        {listHasOverflow ? addSection : null}
      </div>
    </Panel>
  );
}
